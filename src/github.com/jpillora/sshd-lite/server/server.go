package sshd

import (
	"bytes"
	"fmt"
	log "github.com"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/kr/pty"
	"golang.org/x/crypto/ssh"
)

//Server is a simple SSH Daemon
type Server struct {
	config  *Config
	sconfig *ssh.ServerConfig
}

//NewServer creates a new Server
func NewServer(config *Config) (*Server, error) {

	sconfig := &ssh.ServerConfig{}
	server := &Server{config: config, sconfig: sconfig}

	if exec.Command("bash").Run() != nil {
		if exec.Command("sh").Run() != nil {
			return nil, fmt.Errorf("Failed to find shell: %server", config.Shell)
		} else {
			config.Shell = "sh"
		}
	} else {
		config.Shell = "bash"
	}
	server.debugf("Using shell '%s'", config.Shell)

	var private ssh.Signer
	if config.KeyFile != "" {
		//user provided key (can generate with 'ssh-keygen -t rsa')
		privateBytes, err := ioutil.ReadFile(config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to load keyfile(%s)", config.KeyFile)
		}
		private, err = ssh.ParsePrivateKey(privateBytes)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse private key(keyfile:%s)", config.KeyFile)
		}
		log.Log.Info("Key from file %s", config.KeyFile)
	} else {
		//generate key now
		privateBytes, err := generateKey(config.KeySeed)
		if err != nil {
			return nil, fmt.Errorf("Failed to generate private key")
		}
		private, err = ssh.ParsePrivateKey(privateBytes)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse private key")
		}
		if config.KeySeed == "" {
			log.Log.Info("Key from system rng")
		} else {
			log.Log.Info("Key from seed")
		}
	}

	sconfig.AddHostKey(private)
	log.Log.Info("Fingerprint %s", fingerprint(private.PublicKey()))

	//setup auth
	if strings.Contains(config.AuthType, ":") {
		pair := strings.SplitN(config.AuthType, ":", 2)
		u := pair[0]
		p := pair[1]
		sconfig.PasswordCallback = func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if conn.User() == u && string(pass) == p {
				server.debugf("User '%s' authenticated with password", u)
				return nil, nil
			}
			server.debugf("Authentication failed '%s:%s'", conn.User(), pass)
			return nil, fmt.Errorf("denied")
		}
		log.Log.Info("Authentication enabled (user '%s')", u)
	} else if config.AuthType != "" {

		//initial key parse
		keys, last, err := server.parseAuth(time.Time{})
		if err != nil {
			return nil, err
		}

		//setup checker
		sconfig.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {

			//update keys
			if ks, t, err := server.parseAuth(last); err == nil {
				keys = ks
				last = t
				server.debugf("Updated authorized keys")
			}

			k := string(key.Marshal())
			if cmt, exists := keys[k]; exists {
				server.debugf("User '%s' authenticated with public key %s", cmt, fingerprint(key))
				return nil, nil
			}
			server.debugf("User authentication failed with public key %s", fingerprint(key))
			return nil, fmt.Errorf("denied")
		}
		log.Log.Info("Authentication enabled (public keys #%d)", len(keys))
	} else {
		return nil, fmt.Errorf("Missing auth-type")
	}

	return server, nil
}

//Starts listening on port
func (server *Server) Start() error {
	h := server.config.Host
	p := server.config.Port
	var listener net.Listener
	var err error

	//listen
	listener, err = net.Listen("tcp", h+":"+p)
	if err != nil {
		return fmt.Errorf("Failed to listen on %v", p)
	}

	// Accept all connections
	log.Log.Info("Listening on %s:%s...", h, p)
	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Log.Info("Failed to accept incoming connection (%s)", err)
			continue
		}
		//todo(wuheng):socket超时断开（或者判断客户端已断开）
		tcpConn.SetReadDeadline(time.Now().Add(time.Duration(300) * time.Second))
		// Before use, a handshake must be performed on the incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, server.sconfig)
		if err != nil {
			if err != io.EOF {
				log.Log.Info("Failed to handshake (%s)", err)
			}
			continue
		}

		server.debugf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
		// Discard all global out-of-band Requests
		go ssh.DiscardRequests(reqs)
		// Accept all channels
		go server.handleChannels(chans)
	}
}

func (server *Server) handleChannels(chans <-chan ssh.NewChannel) {
	// Service the incoming Channel channel in go routine
	for newChannel := range chans {
		go server.handleChannel(newChannel)
	}
}

func (server *Server) handleChannel(newChannel ssh.NewChannel) {
	// Since we're handling the execution of a shell, we expect a
	// channel type of "session". However, there are also: "x11", "direct-tcpip"
	// and "forwarded-tcpip" channel types.
	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		server.debugf("Could not accept channel (%s)", err)
		return
	}

	shell := exec.Command(server.config.Shell)
	// Allocate a terminal for this channel
	shellf, err := pty.Start(shell)
	close := func() {
		channel.Close()

		delay := true
		//强制结束进程,默认shell会清理进程组
		go func() {
			time.Sleep(time.Second * 3)
			if delay {
				log.Log.Info("shell killed!!!")
				shell.Process.Kill()
			}
		}()
		_, err := shell.Process.Wait()
		if err != nil {
			log.Log.Info("Failed to exit shell (%s)", err)
			shell.Process.Release()
		}
		delay = false
		server.debugf("Session closed")
	}
	if err != nil {
		server.debugf("Could not start pty (%s)", err)
		close()
		return
	}

	//pipe session to shell and visa-versa
	var once sync.Once
	go func() {
		//todo(wuheng): socket断开无法被处理
		io.Copy(channel, shellf)
		once.Do(close)
	}()
	go func() {
		io.Copy(shellf, channel)
		once.Do(close)
	}()

	// Sessions have out-of-band requests such as "shell", "pty-req" and "env"
	go func(in <-chan *ssh.Request) {
		for req := range in {
			ok := false
			switch req.Type {
			case "shell":
				// We only accept the default shell
				// (i.e. no command in the Payload)
				if len(req.Payload) == 0 {
					ok = true
				}
			case "pty-req":
				// Responding true (OK) here will let the client
				// know we have a pty ready for input
				ok = true
				// Parse body...
				termLen := req.Payload[3]
				termEnv := string(req.Payload[4 : termLen+4])
				w, h := parseDims(req.Payload[termLen+4:])
				SetWinsize(shellf.Fd(), w, h)
				log.Log.Info("pty-req '%s'", termEnv)
			case "window-change":
				w, h := parseDims(req.Payload)
				SetWinsize(shellf.Fd(), w, h)
				continue //no response
			case "exec":
				//putty need this
				ok = true
			}
			if !ok {
				log.Log.Info("declining %s request???", req.Type)
			}
			req.Reply(ok, nil)
		}
	}(requests)
}

func (server *Server) parseAuth(last time.Time) (map[string]string, time.Time, error) {

	info, err := os.Stat(server.config.AuthType)
	if err != nil {
		return nil, last, fmt.Errorf("Missing auth keys file")
	}

	t := info.ModTime()
	if t.Before(last) || t == last {
		return nil, last, fmt.Errorf("Not updated")
	}

	//grab file
	b, _ := ioutil.ReadFile(server.config.AuthType)
	lines := bytes.Split(b, []byte("\n"))
	//parse each line
	keys := map[string]string{}
	for _, l := range lines {
		if key, cmt, _, _, err := ssh.ParseAuthorizedKey(l); err == nil {
			keys[string(key.Marshal())] = cmt
		}
	}
	//ensure we got something
	if len(keys) == 0 {
		return nil, last, fmt.Errorf("No keys found in %s", server.config.AuthType)
	}
	return keys, t, nil
}

func (server *Server) debugf(f string, args ...interface{}) {
	if server.config.LogVerbose {
		log.Log.Info(f, args...)
	}
}
