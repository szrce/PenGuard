package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

var blacklistdir = "/usr/local/etc/penguard/blacklist/"
var	bdir =  "/usr/local/sbin"
var blacklist = make(map[string]struct{})
var DEBUG = false

var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 32*2048)
    },
}

func copyData(dst io.Writer, src io.Reader) {
    buffer := bufferPool.Get().([]byte)
    defer bufferPool.Put(buffer)
    io.CopyBuffer(dst, src, buffer)
}

func main() {

	ls, err := net.Listen("tcp4", ":9020")
	fmt.Println("listen:9020")
	loadBlacklist()
	if err != nil {
		panic(err)
	}

	for {
		conn, err := ls.Accept()
		if err != nil {
			fmt.Println("connect failed", err)
		}
		go handler(conn)
	}

}

func loadBlacklist() {
    file, err := os.Open(blacklistdir + "sitelist")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
    for scanner.Scan() {
        blacklist[scanner.Text()] = struct{}{}
    }
}

func checkdomain(domain string) bool {
	if DEBUG {
		fmt.Println("check ->:" +  domain)
	}
    _, exists := blacklist[domain]
    return exists
}

func isHTTPs(val string) bool {
	//var part = strings.Split(val, ":")
	if val == "CONNECT" {
		return true
	}
	return false
}
func isBlocked(val string) bool{
	if checkdomain(val) {
		return true
	}
	return false
}


func handler(conn net.Conn) {
	if DEBUG {
		fmt.Printf("======>connection coming this  %s \n\n", conn.RemoteAddr().String())
	}
	for {

		buf := make([]byte, 1024)
		_, err := conn.Read(buf[:])


		if err != nil {
			if DEBUG {
				fmt.Printf("\nconnection broken %s \n", err)
			}
			conn.Close()
			break
		}

		requestStr := string(buf)
		requestParts := strings.Split(requestStr, " ")
		requrl, err := url.Parse(requestParts[1])
		
		
		if isHTTPs(requestParts[0]){
			//connn

			var part = strings.Split(requestParts[1], ":")
			if isBlocked(part[0]) {
				conn.Write([]byte("<html><body><div style=\"background-color:red;\">blocked 443</div></body></html>"))
				conn.Close()
			}

			//check the address exist?
			CheckConn, err := net.Dial("tcp",requestParts[1])
			if err != nil {
				// handle error
			}
			conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
			go copyData(conn, CheckConn)
			copyData(CheckConn, conn)
			

		}else{
			if DEBUG {
				fmt.Println(requrl)
			}
			if isBlocked(requrl.Host) {
				conn.Write([]byte("<html><body><div style=\"background-color:red;\">blocked</div></body></html>"))
				conn.Close()
			} else {
				resp, err := http.Get("http://"+requrl.Host + requrl.Path)
				if err != nil {
				}
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				conn.Write(body)
				conn.Close()
			}

		}

	}

}
