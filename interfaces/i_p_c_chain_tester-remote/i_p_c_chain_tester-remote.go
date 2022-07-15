// Code generated by Thrift Compiler (0.15.0). DO NOT EDIT.

package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	thrift "github.com/apache/thrift/lib/go/thrift"
	"interfaces"
)

var _ = interfaces.GoUnusedProtection__

func Usage() {
  fmt.Fprintln(os.Stderr, "Usage of ", os.Args[0], " [-h host:port] [-u url] [-f[ramed]] function [arg1 [arg2...]]:")
  flag.PrintDefaults()
  fmt.Fprintln(os.Stderr, "\nFunctions:")
  fmt.Fprintln(os.Stderr, "  void init_vm_api()")
  fmt.Fprintln(os.Stderr, "  void init_apply_request()")
  fmt.Fprintln(os.Stderr, "  string pack_abi(string abi)")
  fmt.Fprintln(os.Stderr, "  string pack_action_args(i32 id, string contract, string action, string action_args)")
  fmt.Fprintln(os.Stderr, "  string unpack_action_args(i32 id, string contract, string action, string raw_args)")
  fmt.Fprintln(os.Stderr, "  i32 new_chain()")
  fmt.Fprintln(os.Stderr, "  i32 free_chain(i32 id)")
  fmt.Fprintln(os.Stderr, "  void produce_block(i32 id)")
  fmt.Fprintln(os.Stderr, "  string push_action(i32 id, string account, string action, string arguments, string permissions)")
  fmt.Fprintln(os.Stderr, "  string push_actions(i32 id,  actions)")
  fmt.Fprintln(os.Stderr)
  os.Exit(0)
}

type httpHeaders map[string]string

func (h httpHeaders) String() string {
  var m map[string]string = h
  return fmt.Sprintf("%s", m)
}

func (h httpHeaders) Set(value string) error {
  parts := strings.Split(value, ": ")
  if len(parts) != 2 {
    return fmt.Errorf("header should be of format 'Key: Value'")
  }
  h[parts[0]] = parts[1]
  return nil
}

func main() {
  flag.Usage = Usage
  var host string
  var port int
  var protocol string
  var urlString string
  var framed bool
  var useHttp bool
  headers := make(httpHeaders)
  var parsedUrl *url.URL
  var trans thrift.TTransport
  _ = strconv.Atoi
  _ = math.Abs
  flag.Usage = Usage
  flag.StringVar(&host, "h", "localhost", "Specify host and port")
  flag.IntVar(&port, "p", 9090, "Specify port")
  flag.StringVar(&protocol, "P", "binary", "Specify the protocol (binary, compact, simplejson, json)")
  flag.StringVar(&urlString, "u", "", "Specify the url")
  flag.BoolVar(&framed, "framed", false, "Use framed transport")
  flag.BoolVar(&useHttp, "http", false, "Use http")
  flag.Var(headers, "H", "Headers to set on the http(s) request (e.g. -H \"Key: Value\")")
  flag.Parse()
  
  if len(urlString) > 0 {
    var err error
    parsedUrl, err = url.Parse(urlString)
    if err != nil {
      fmt.Fprintln(os.Stderr, "Error parsing URL: ", err)
      flag.Usage()
    }
    host = parsedUrl.Host
    useHttp = len(parsedUrl.Scheme) <= 0 || parsedUrl.Scheme == "http" || parsedUrl.Scheme == "https"
  } else if useHttp {
    _, err := url.Parse(fmt.Sprint("http://", host, ":", port))
    if err != nil {
      fmt.Fprintln(os.Stderr, "Error parsing URL: ", err)
      flag.Usage()
    }
  }
  
  cmd := flag.Arg(0)
  var err error
  var cfg *thrift.TConfiguration = nil
  if useHttp {
    trans, err = thrift.NewTHttpClient(parsedUrl.String())
    if len(headers) > 0 {
      httptrans := trans.(*thrift.THttpClient)
      for key, value := range headers {
        httptrans.SetHeader(key, value)
      }
    }
  } else {
    portStr := fmt.Sprint(port)
    if strings.Contains(host, ":") {
           host, portStr, err = net.SplitHostPort(host)
           if err != nil {
                   fmt.Fprintln(os.Stderr, "error with host:", err)
                   os.Exit(1)
           }
    }
    trans = thrift.NewTSocketConf(net.JoinHostPort(host, portStr), cfg)
    if err != nil {
      fmt.Fprintln(os.Stderr, "error resolving address:", err)
      os.Exit(1)
    }
    if framed {
      trans = thrift.NewTFramedTransportConf(trans, cfg)
    }
  }
  if err != nil {
    fmt.Fprintln(os.Stderr, "Error creating transport", err)
    os.Exit(1)
  }
  defer trans.Close()
  var protocolFactory thrift.TProtocolFactory
  switch protocol {
  case "compact":
    protocolFactory = thrift.NewTCompactProtocolFactoryConf(cfg)
    break
  case "simplejson":
    protocolFactory = thrift.NewTSimpleJSONProtocolFactoryConf(cfg)
    break
  case "json":
    protocolFactory = thrift.NewTJSONProtocolFactory()
    break
  case "binary", "":
    protocolFactory = thrift.NewTBinaryProtocolFactoryConf(cfg)
    break
  default:
    fmt.Fprintln(os.Stderr, "Invalid protocol specified: ", protocol)
    Usage()
    os.Exit(1)
  }
  iprot := protocolFactory.GetProtocol(trans)
  oprot := protocolFactory.GetProtocol(trans)
  client := interfaces.NewIPCChainTesterClient(thrift.NewTStandardClient(iprot, oprot))
  if err := trans.Open(); err != nil {
    fmt.Fprintln(os.Stderr, "Error opening socket to ", host, ":", port, " ", err)
    os.Exit(1)
  }
  
  switch cmd {
  case "init_vm_api":
    if flag.NArg() - 1 != 0 {
      fmt.Fprintln(os.Stderr, "InitVMAPI requires 0 args")
      flag.Usage()
    }
    fmt.Print(client.InitVMAPI(context.Background()))
    fmt.Print("\n")
    break
  case "init_apply_request":
    if flag.NArg() - 1 != 0 {
      fmt.Fprintln(os.Stderr, "InitApplyRequest requires 0 args")
      flag.Usage()
    }
    fmt.Print(client.InitApplyRequest(context.Background()))
    fmt.Print("\n")
    break
  case "pack_abi":
    if flag.NArg() - 1 != 1 {
      fmt.Fprintln(os.Stderr, "PackAbi requires 1 args")
      flag.Usage()
    }
    argvalue0 := flag.Arg(1)
    value0 := argvalue0
    fmt.Print(client.PackAbi(context.Background(), value0))
    fmt.Print("\n")
    break
  case "pack_action_args":
    if flag.NArg() - 1 != 4 {
      fmt.Fprintln(os.Stderr, "PackActionArgs_ requires 4 args")
      flag.Usage()
    }
    tmp0, err30 := (strconv.Atoi(flag.Arg(1)))
    if err30 != nil {
      Usage()
      return
    }
    argvalue0 := int32(tmp0)
    value0 := argvalue0
    argvalue1 := flag.Arg(2)
    value1 := argvalue1
    argvalue2 := flag.Arg(3)
    value2 := argvalue2
    argvalue3 := flag.Arg(4)
    value3 := argvalue3
    fmt.Print(client.PackActionArgs_(context.Background(), value0, value1, value2, value3))
    fmt.Print("\n")
    break
  case "unpack_action_args":
    if flag.NArg() - 1 != 4 {
      fmt.Fprintln(os.Stderr, "UnpackActionArgs_ requires 4 args")
      flag.Usage()
    }
    tmp0, err34 := (strconv.Atoi(flag.Arg(1)))
    if err34 != nil {
      Usage()
      return
    }
    argvalue0 := int32(tmp0)
    value0 := argvalue0
    argvalue1 := flag.Arg(2)
    value1 := argvalue1
    argvalue2 := flag.Arg(3)
    value2 := argvalue2
    argvalue3 := []byte(flag.Arg(4))
    value3 := argvalue3
    fmt.Print(client.UnpackActionArgs_(context.Background(), value0, value1, value2, value3))
    fmt.Print("\n")
    break
  case "new_chain":
    if flag.NArg() - 1 != 0 {
      fmt.Fprintln(os.Stderr, "NewChain_ requires 0 args")
      flag.Usage()
    }
    fmt.Print(client.NewChain_(context.Background()))
    fmt.Print("\n")
    break
  case "free_chain":
    if flag.NArg() - 1 != 1 {
      fmt.Fprintln(os.Stderr, "FreeChain requires 1 args")
      flag.Usage()
    }
    tmp0, err38 := (strconv.Atoi(flag.Arg(1)))
    if err38 != nil {
      Usage()
      return
    }
    argvalue0 := int32(tmp0)
    value0 := argvalue0
    fmt.Print(client.FreeChain(context.Background(), value0))
    fmt.Print("\n")
    break
  case "produce_block":
    if flag.NArg() - 1 != 1 {
      fmt.Fprintln(os.Stderr, "ProduceBlock requires 1 args")
      flag.Usage()
    }
    tmp0, err39 := (strconv.Atoi(flag.Arg(1)))
    if err39 != nil {
      Usage()
      return
    }
    argvalue0 := int32(tmp0)
    value0 := argvalue0
    fmt.Print(client.ProduceBlock(context.Background(), value0))
    fmt.Print("\n")
    break
  case "push_action":
    if flag.NArg() - 1 != 5 {
      fmt.Fprintln(os.Stderr, "PushAction requires 5 args")
      flag.Usage()
    }
    tmp0, err40 := (strconv.Atoi(flag.Arg(1)))
    if err40 != nil {
      Usage()
      return
    }
    argvalue0 := int32(tmp0)
    value0 := argvalue0
    argvalue1 := flag.Arg(2)
    value1 := argvalue1
    argvalue2 := flag.Arg(3)
    value2 := argvalue2
    argvalue3 := flag.Arg(4)
    value3 := argvalue3
    argvalue4 := flag.Arg(5)
    value4 := argvalue4
    fmt.Print(client.PushAction(context.Background(), value0, value1, value2, value3, value4))
    fmt.Print("\n")
    break
  case "push_actions":
    if flag.NArg() - 1 != 2 {
      fmt.Fprintln(os.Stderr, "PushActions requires 2 args")
      flag.Usage()
    }
    tmp0, err45 := (strconv.Atoi(flag.Arg(1)))
    if err45 != nil {
      Usage()
      return
    }
    argvalue0 := int32(tmp0)
    value0 := argvalue0
    arg46 := flag.Arg(2)
    mbTrans47 := thrift.NewTMemoryBufferLen(len(arg46))
    defer mbTrans47.Close()
    _, err48 := mbTrans47.WriteString(arg46)
    if err48 != nil { 
      Usage()
      return
    }
    factory49 := thrift.NewTJSONProtocolFactory()
    jsProt50 := factory49.GetProtocol(mbTrans47)
    containerStruct1 := interfaces.NewIPCChainTesterPushActionsArgs()
    err51 := containerStruct1.ReadField2(context.Background(), jsProt50)
    if err51 != nil {
      Usage()
      return
    }
    argvalue1 := containerStruct1.Actions
    value1 := argvalue1
    fmt.Print(client.PushActions(context.Background(), value0, value1))
    fmt.Print("\n")
    break
  case "":
    Usage()
    break
  default:
    fmt.Fprintln(os.Stderr, "Invalid function ", cmd)
  }
}
