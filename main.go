package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/ssh"
)

type Forward struct {
	LocalIP    string
	RemoteIP   string
	LocalPort  int
	RemotePort int
	Enable     bool
}

type Config struct {
	Host       string
	Port       int
	User       string
	Password   string
	PrivateKey string
	Forward    []Forward
}

var debug = flag.Bool("debug", false, "debug mode")
var configfile = flag.String("config", "./config.json", "config file")

var config Config
var PrivateKey []byte
var sshConfig *ssh.ClientConfig

func main() {
	flag.Parse()

	wd, err := os.Executable()
	if err != nil {
		log.Fatalln("Executable", err)
	}

	if *debug {
		log.Println("Executable", wd)
		log.Println("chdir", filepath.Dir(wd))
	}

	err = os.Chdir(filepath.Dir(wd))
	if err != nil {
		log.Fatalln("Chdir", err)
	}

	initConfig()
	initSSH()

	for _, item := range config.Forward {
		if item.Enable {
			go listen(item)
		}
	}

	end := make(chan int)
	<-end
}

func listen(item Forward) {
	localAddress := item.LocalIP + ":" + strconv.Itoa(item.LocalPort)

	l, err := net.Listen("tcp", localAddress)
	if err != nil {
		log.Fatalln("Net Listen", err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		go func(c net.Conn) {
			defer c.Close()

			remoteAddress := item.RemoteIP + ":" + strconv.Itoa(item.RemotePort)

			log.Println("forward:", localAddress, "=>", remoteAddress)

			to, err := sshDial(remoteAddress)
			if err != nil {
				log.Println(err)
			}

			defer sshDialClose(to)

			if *debug {
				defer func() {
					log.Println("close:", localAddress, "=>", remoteAddress)
				}()
			}

			err = relay(c, to)
			if err != nil {
				log.Println(err)
			}
		}(conn)
	}

}

var connNum = 0
var server *ssh.Client

func sshDial(to string) (net.Conn, error) {
	var err error

	if server == nil {
		address := config.Host + ":" + strconv.Itoa(config.Port)

		if *debug {
			log.Println("ssh Dial", address)
		}

		server, err = ssh.Dial("tcp", address, sshConfig)
		if err != nil {
			return nil, err
		}
	}

	c, err := server.Dial("tcp", to)
	if err != nil {
		return nil, err
	}

	connNum++

	return c, err
}

func sshDialClose(c net.Conn) {
	c.Close()

	connNum--

	if connNum < 1 {
		if server != nil {
			if *debug {
				log.Println("ssh close")
			}

			server.Close()
			server = nil
		}
	}
}

func initSSH() {
	if PrivateKey == nil {
		sshConfig = &ssh.ClientConfig{
			User: config.User,
			Auth: []ssh.AuthMethod{
				ssh.Password(config.Password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	} else {
		signer, err := ssh.ParsePrivateKey(PrivateKey)
		if err != nil {
			log.Fatalln("unable to parse private key:", err)
		}

		sshConfig = &ssh.ClientConfig{
			User: config.User,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	}

}

func relay(a, b net.Conn) (err error) {
	if t, ok := a.(*net.TCPConn); ok {
		t.SetNoDelay(true)
		t.SetKeepAlive(true)
	}
	if t, ok := b.(*net.TCPConn); ok {
		t.SetNoDelay(true)
		t.SetKeepAlive(true)
	}

	ch := make(chan error, 1)

	go func() {
		_, e := io.Copy(a, b)
		ch <- e
	}()
	go func() {
		_, e := io.Copy(b, a)
		ch <- e
	}()

	//first err
	return <-ch
}

func initConfig() {
	data, err := ioutil.ReadFile(*configfile)
	if err != nil {
		log.Fatalln("config", err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatalln("config", err)
	}

	if config.PrivateKey != "" {
		data, err := ioutil.ReadFile(config.PrivateKey)
		if err != nil {
			log.Fatalln("PrivateKey", err)
		}
		PrivateKey = data
	}
}
