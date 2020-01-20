package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/cog"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/sign-go"
	"github.com/valyala/gorpc"
)

var (
	version = "[manual build]"
	usage   = "glod " + version + `


Usage:
  glod [options] server [-s <sock>]
  glod [options] [-s <sock>] set <key> <value> [-p <pid>]
  glod [options] [-s <sock>] get <key>
  glod [options] [-s <sock>] list
  glod -h | --help
  glod --version

Options:
  -p --pid <pid>      Set owner of data, if pid dies, data flushes.
  -s --socket <sock>  Set path to socket file. [default: /var/run/user/$UID/glod.sock]
  -h --help           Show this screen.
  --version           Show version.
`
)

var log *cog.Logger

func init() {
	stderr := lorg.NewLog()
	stderr.SetIndentLines(true)
	stderr.SetFormat(
		lorg.NewFormat("${time} ${level:[%s]:right:short} ${prefix}%s"),
	)

	log = cog.NewLogger(stderr)
}

func main() {
	usage := strings.Replace(usage, "$UID", fmt.Sprint(os.Getuid()), -1)

	opts, err := docopt.ParseArgs(usage, nil, version)
	if err != nil {
		panic(err)
	}

	var args struct {
		Server bool
		Socket string
		Get    bool
		Set    bool
		List   bool
		Key    string
		Value  string
		PID    int `docopt:"--pid"`
	}

	err = opts.Bind(&args)
	if err != nil {
		panic(err)
	}

	if args.Server {
		err = runServer(args.Socket)
		if err != nil {
			log.Fatalf(err, "unable to run server at %s", args.Socket)
		}

		return
	}

	sockClient := gorpc.NewUnixClient(args.Socket)
	sockClient.RequestTimeout = time.Second * 2
	sockClient.Start()
	defer sockClient.Stop()

	dispatcher := NewServiceDispatcher(&Service{})
	client := dispatcher.NewFuncClient(sockClient)

	if args.Set {
		_, err := client.Call("set", &Item{
			Key:   args.Key,
			Value: args.Value,
			PID:   args.PID,
		})
		if err != nil {
			log.Fatalf(err, "unable to set value for given key")
		}

		return
	}

	if args.Get {
		response, err := client.Call("get", args.Key)
		if err != nil {
			log.Fatalf(err, "unable to get value for given key")
		}

		if response != "" {
			fmt.Println(response)
		}

		return
	}

	if args.List {
		response, err := client.Call("list", false)
		if err != nil {
			log.Fatalf(err, "unable to list keys")
		}

		if response != nil {
			for _, item := range response.([]*Item) {
				fmt.Printf("%s %s\n", item.Key, item.Value)
			}
		}

		return
	}
}

func runServer(socketPath string) error {
	service := NewService()

	dispatcher := NewServiceDispatcher(service)

	server := gorpc.NewUnixServer(socketPath, dispatcher.NewHandlerFunc())

	err := server.Start()
	if err != nil {
		return karma.Format(
			err,
			"unable to start rpc server",
		)
	}

	sign.Notify(func(_ os.Signal) bool {
		return false
	}, syscall.SIGINT, syscall.SIGTERM)

	server.Stop()

	return nil
}
