package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"encoding/base64"
	"encoding/json"

	"github.com/docker/docker/pkg/mount"
)

//Example of a json payload received by this script (k8s 1.15+)
//{
//  "kubernetes.io/mounterArgs.FsGroup": "33",
//  "kubernetes.io/fsType": "",
//  "kubernetes.io/pod.name": "nginx-deployment-549ddfb5fc-rnqk8",
//  "kubernetes.io/pod.namespace": "default",
//  "kubernetes.io/pod.uid": "bb6b2e46-c80d-4c86-920c-8e08736fa211",
//  "kubernetes.io/pvOrVolumeName": "test-volume",
//  "kubernetes.io/readwrite": "rw",
//  "kubernetes.io/serviceAccount.name": "default",
//  "opts": "domain=Foo",
//  "server": "fooserver123",
//  "share": "/test"
//}

//k8s versions prior to 1.15 pass fsGroup differently
//{
//-  "kubernetes.io/mounterArgs.FsGroup": "33",
//+  "kubernetes.io/fsGroup": "33",
//  ...
//}

// JSONArguments is the data coming in from cli
type JSONArguments struct {
	MounterFsGroup     string `json:"kubernetes.io/mounterArgs.FsGroup"`
	FsGroup            string `json:"kubernetes.io/fsGroup"`
	FsType             string `json:"kubernetes.io/fsType"`
	PodName            string `json:"kubernetes.io/pod.name"`
	PodNamespace       string `json:"kubernetes.io/pod.namespace"`
	PodUID             string `json:"kubernetes.io/pod.uid"`
	PVOrVolumeName     string `json:"kubernetes.io/pvOrVolumeName"`
	ReadWrite          string `json:"kubernetes.io/readwrite"`
	ServiceAccountName string `json:"kubernetes.io/kubernetes.io/serviceAccount.name"`

	Domain   string `json:"kubernetes.io/secret/domain"`
	Password string `json:"kubernetes.io/secret/password"`
	Username string `json:"kubernetes.io/secret/username"`

	Opts   string `json:"opts"`
	Server string `json:"server"`
	Share  string `json:"share"`
}

// JSONError is any error that happened in the system
type JSONError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// JSONSuccess yay
type JSONSuccess struct {
}

// exit code handler
func handleExit() {
	if e := recover(); e != nil {
		if err, ok := e.(JSONError); ok == true {
			b, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				panic(marshalErr)
			}
			os.Stderr.Write(b)
			os.Exit(1)
		}
		if _, ok := e.(JSONSuccess); ok == true {
			os.Stderr.WriteString("{\"status\": \"Success\"}")
			os.Exit(0)
		}
		panic(e) // not an Exit, bubble up
	}
}

func main() {
	defer handleExit()
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}
	if args[0] == "init" {
		os.Stderr.WriteString("{\"status\": \"Success\", \"capabilities\": {\"attach\": false}}")
		os.Exit(0)
	}

	if args[0] == "mount" {
		if len(args) < 2 {
			usage()
		}
		domount(args[1], readJSON(args[2]))
	}

	if args[0] == "unmount" {
		unmount(args[1])
	}

	usage()
}

func usage() {
	fmt.Fprintf(os.Stderr, "Invalid usage. Usage: \n")
	fmt.Fprintf(os.Stderr, "\t%s init\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\t%s mount <mount dir> <json params>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\t%s unmount <mount dir>\n", os.Args[0])
	os.Exit(1)
}

func readJSON(jsonParams string) JSONArguments {
	args := JSONArguments{
		ReadWrite: "RW",
	}
	err := json.Unmarshal([]byte(jsonParams), &args)
	if err != nil {
		handleError(err)
	}
	// k8s versions prior to 1.15 pass fsGroup differently
	if args.FsGroup == "" {
		args.FsGroup = args.MounterFsGroup
	}
	if args.ReadWrite == "" {
		args.ReadWrite = "RW"
	}
	if args.Domain != "" {
		decoded, err := base64.StdEncoding.DecodeString(args.Domain)
		if err != nil {
			handleError(err)
		}
		args.Domain = string(decoded)
	}
	if args.Username != "" {
		decoded, err := base64.StdEncoding.DecodeString(args.Username)
		if err != nil {
			handleError(err)
		}
		args.Username = string(decoded)
	}
	if args.Password != "" {
		decoded, err := base64.StdEncoding.DecodeString(args.Password)
		if err != nil {
			handleError(err)
		}
		args.Password = string(decoded)
	}
	return args
}

func handleError(err error) {
	panic(JSONError{
		Status:  "Failed",
		Message: fmt.Sprintf("%s", err),
	})
}

func writeCredentials(credentialsPath string, Domain string, Username string, Password string) {
	f, err := os.Create("/tmp/dat2")
	if err != nil {
		handleError(err)
	}
	defer f.Close()

	if Username != "" {
		_, err := f.WriteString(fmt.Sprintf("username=%s\n", Username))
		if err != nil {
			handleError(err)
		}
	}
	if Domain != "" {
		_, err := f.WriteString(fmt.Sprintf("domain=%s\n", Domain))
		if err != nil {
			handleError(err)
		}
	}
	if Password != "" {
		_, err := f.WriteString(fmt.Sprintf("password=%s\n", Password))
		if err != nil {
			handleError(err)
		}
	}

	err = f.Sync()
	if err != nil {
		handleError(err)
	}
}

func domount(mntPath string, args JSONArguments) {
	var finalOpts []string
	credentialsPath := fmt.Sprintf("/tmp/temporary.%d.tmp", args.PodUID)

	isMounted, err := mount.Mounted(mntPath)
	if err != nil {
		handleError(err)
	}

	// already mounted so just return success
	if isMounted {
		panic(JSONSuccess{})
	}

	// Complex passwords with symbols and special characters
	// are a painful to deal with in a shell script.
	// This works around that by saving credentials in a
	// temporary file.
	finalOpts = append(finalOpts, fmt.Sprintf("credentials=%s", credentialsPath))

	if args.Opts != "" {
		finalOpts = append(finalOpts, strings.Split(args.Opts, ",")...)
	}
	finalOpts = append(finalOpts, args.ReadWrite)

	if args.FsGroup != "" {
		finalOpts = append(finalOpts, fmt.Sprintf("uid=%s,gid=%s", args.FsGroup, args.FsGroup))
	}

	// make sure the mount path exists
	_ = os.Mkdir(mntPath, os.ModePerm)

	// write out credentials
	writeCredentials(credentialsPath, args.Domain, args.Username, args.Password)
	// remove it when we are done
	defer os.Remove(credentialsPath)

	err = syscall.Mount(mntPath, fmt.Sprintf("//%s%s", args.Server, args.Share), "cifs", 0, strings.Join(finalOpts, ","))
	if err != nil {
		// TODO - \"message\": \"domain=${DOMAIN} username=${USERNAME} Failed returncode=$R mount -t cifs -o $FINALOPTS \"//$CIFS_SERVER$CIFS_SHARE\" \"$MNTPATH\"\"}"
		handleError(err)
	}

	panic(JSONSuccess{})
}

func unmount(mntPath string) {

	isMounted, err := mount.Mounted(mntPath)
	if err != nil {
		handleError(err)
	}

	// already mounted so just return success
	if !isMounted {
		panic(JSONSuccess{})
	}

	err = syscall.Unmount(mntPath, 0)
	if err != nil {
		handleError(err)
	}

	panic(JSONSuccess{})
}
