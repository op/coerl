// coerl colourises curl's output.
package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
)

const (
	reset       = "\x1b[0m"
	green       = "\x1b[32m"
	yellow      = "\x1b[33m"
	blue        = "\x1b[34m"
	magenta     = "\x1b[35m"
	cyan        = "\x1b[36m"
	white       = "\x1b[37m"
	grayscale14 = "\x1b[38;5;248m"
	grayscale16 = "\x1b[38;5;250m"
)

func main() {
	var (
		// command to execute
		arg0 = getEnv("COERL_CURL_BIN", "curl")

		// > -- header out
		headerOut = getEnv("COERL_HEADER_OUT", grayscale16)
		// < -- header in
		headerIn = getEnv("COERL_HEADER_IN", grayscale16)
		// } -- data out
		dataOut = getEnv("COERL_DATA_OUT", white)
		// { -- (ssl) data in
		dataIn = getEnv("COERL_DATA_IN", white)
		// } -- SSL data out
		sslDataOut = getEnv("COERL_SSL_DATA_OUT", yellow)
		// { -- SSL data in
		sslDataIn = getEnv("COERL_SSL_DATA_IN", yellow)
		// * -- text
		text = getEnv("COERL_TEXT", magenta)

		// other knobs
		headerOff    = getBoolEnv("COERL_HEADER_OFF", false)
		headerOutOff = getBoolEnv("COERL_HEADER_OUT_OFF", headerOff)
		headerInOff  = getBoolEnv("COERL_HEADER_IN_OFF", headerOff)

		dataOff    = getBoolEnv("COERL_DATA_OFF", false)
		dataOutOff = getBoolEnv("COERL_DATA_OUT_OFF", dataOff)
		dataInOff  = getBoolEnv("COERL_DATA_IN_OFF", dataOff)

		sslDataOff    = getBoolEnv("COERL_SSL_DATA_OFF", dataOff)
		sslDataOutOff = getBoolEnv("COERL_SSL_DATA_OUT_OFF", sslDataOff)
		sslDataInOff  = getBoolEnv("COERL_SSL_DATA_IN_OFF", sslDataOff)

		textOff = getBoolEnv("COERL_TEXT_OFF", false)
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, arg0, os.Args[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	connected := false

	s := bufio.NewScanner(io.MultiReader(stderr))
	for s.Scan() {
		b := s.Bytes()
		if len(b) > 2 {
			switch b[0] {
			case '<':
				connected = true
				if headerInOff {
					continue
				}
				os.Stderr.WriteString(headerIn)
			case '{':
				if connected {
					if dataInOff {
						continue
					}
					os.Stderr.WriteString(dataIn)
				} else {
					if sslDataInOff {
						continue
					}
					os.Stderr.WriteString(sslDataIn)
				}
			case '>':
				connected = true
				if headerOutOff {
					continue
				}
				os.Stderr.WriteString(headerOut)
			case '}':
				if connected {
					if dataOutOff {
						continue
					}
					os.Stderr.WriteString(dataOut)
				} else {
					if sslDataOutOff {
						continue
					}
					os.Stderr.WriteString(sslDataOut)
				}
			case '*':
				if textOff {
					continue
				}
				os.Stderr.WriteString(text)
			default:
				// TODO: inspect what data this is, seem to be SSL info
				if connected {
					if dataOutOff {
						continue
					}
					os.Stderr.WriteString(dataOut)
				} else {
					if sslDataOutOff {
						continue
					}
					os.Stderr.WriteString(sslDataOut)
				}
			}

			if b[0] == '>' || b[0] == '<' {
				// skip eg '< '
				os.Stderr.Write(b[0:2])
				l := b[2:]

				if httpMethodPrefix(l) {
					i := bytes.IndexByte(l, ' ')
					os.Stderr.WriteString(green)
					os.Stderr.Write(l[:i]) // http method
					os.Stderr.WriteString(cyan)
					os.Stderr.Write(l[i:]) // URL
				} else if bytes.HasPrefix(l, []byte("HTTP/")) {
					i := bytes.IndexByte(l, '/')
					os.Stderr.WriteString(green)
					os.Stderr.Write(l[:i]) // 'HTTP/x.y'
					os.Stderr.WriteString(cyan)
					os.Stderr.Write(l[i:])
				} else if i := bytes.IndexByte(l, ':'); i != -1 {
					os.Stderr.Write(l[:i]) // key
					os.Stderr.Write([]byte(cyan))
					os.Stderr.Write(l[i:]) // value
				} else {
					os.Stderr.Write(l) // fallback
				}
			} else {
				os.Stderr.Write(b)
			}
		}

		os.Stderr.Write([]byte(reset))
		os.Stderr.Write([]byte("\n"))
	}

	io.Copy(os.Stdout, stdout)

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	} else {
		return def
	}
}

func getBoolEnv(key string, def bool) bool {
	if b, err := strconv.ParseBool(getEnv(key, "")); err != nil {
		return def
	} else {
		return b
	}
}

func httpMethodPrefix(b []byte) bool {
	for _, m := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		if bytes.HasPrefix(b, []byte(m+" ")) {
			return true
		}
	}
	return false
}
