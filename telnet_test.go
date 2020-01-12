package telnet

import (
	"bufio"
	"bytes"
	"io"
	"sync"
	"testing"
	"time"
)

type testReadCase struct {
	name        string
	args        [][]byte
	want        []byte
	wantTimeout bool
}

func callWithTimeout(
	tc *TelnetClient,
	payload func() []byte,
) (result []byte, expired bool) {
	doneCh := make(chan bool)

	go func() {
		result = payload()
		doneCh <- true
	}()

	select {
	case _ = <-doneCh:
		break
	case <-time.After(tc.Timeout):
		expired = true
		break
	}

	return
}

func (rc *testReadCase) run(
	t *testing.T,
	tc *TelnetClient,
	payload func() []byte,
) {
	var wg sync.WaitGroup

	r, w := io.Pipe()
	wg.Add(2)

	// server
	go func() {
		for _, a := range rc.args {
			w.Write(a)
		}

		wg.Done()
	}()

	// testable client
	go func() {
		tc.reader = bufio.NewReader(r)
		fact, expired := callWithTimeout(tc, payload)

		if expired != rc.wantTimeout {
			t.Errorf("[%s] unexpected timeout value", rc.name)
		} else if bytes.Compare(fact, rc.want) != 0 {
			t.Errorf(
				"[%s] wrong result:\n\t\tfact = %v\n\t\twant = %v",
				rc.name, fact, rc.want)
		}

		wg.Done()
	}()

	wg.Wait()
}

func Test_skipSBSequence(t *testing.T) {
	tc := &TelnetClient{Timeout: 10 * time.Millisecond}
	tests := []testReadCase{
		{
			name: "skipSBSequence: Just skip SB/SE sequence",
			args: [][]byte{
				[]byte{0xfa, 0x18, 0, 0x56, 0x54, 0x32, 0x32, 0x30, 0xff, 0xf0},
			},
			want: []byte{},
		},
		{
			name: "skipSBSequence: Skip first SB/SE sequence",
			args: [][]byte{
				[]byte{
					0xfa, 0x18, 0, 0x56, 0x54, 0x32, 0x32, 0x30, 0xff, 0xf0,
					0x70, 0x6c, 0x61, 0x69, 0x6e, 0x20, 0x74, 0x65, 0x78, 0x74,
					0xff, 0xfa, 0x18, 0, 0x56, 0x54, 0x32, 0x32, 0x30, 0xff, 0xf0,
				},
			},
			want: []byte{
				0x70, 0x6c, 0x61, 0x69, 0x6e, 0x20, 0x74, 0x65, 0x78, 0x74,
				0xff, 0xfa, 0x18, 0, 0x56, 0x54, 0x32, 0x32, 0x30, 0xff, 0xf0,
			},
		},
		{
			name:        "skipSBSequence error: without SE",
			wantTimeout: true,
			args: [][]byte{
				[]byte{
					0xfa, 0x18, 0, 0x56, 0x54, 0x32, 0x32, 0x30, 0xff, // 0xf0,
					0x70, 0x6c, 0x61, 0x69, 0x6e, 0x20, 0x74, 0x65, 0x78, 0x74,
					0xff, 0xfa, 0x18, 0, 0x56, 0x54, 0x32, 0x32, 0x30, // 0xff, 0xf0,
				},
			},
			want: []byte{},
		},
		{
			name: "skipSBSequence error: without IAC for SE",
			args: [][]byte{
				[]byte{
					0xfa, 0x18, 0, 0x56, 0x54, 0x32, 0x32, 0x30, 0xf0,
					0x70, 0x6c, 0x61, 0x69, 0x6e, 0x20, 0x74, 0x65, 0x78, 0x74,
					0xff, 0xfa, 0x18, 0, 0x56, 0x54, 0x32, 0x32, 0x30, 0xff, 0xf0,
					0x70, 0x6c, 0x61, 0x69, 0x6e, 0x20, 0x74, 0x65, 0x78, 0x74,
				},
			},
			want: []byte{0x70, 0x6c, 0x61, 0x69, 0x6e, 0x20, 0x74, 0x65, 0x78, 0x74},
		},
		{
			name: "skipSBSequence error: multiple IAC",
			args: [][]byte{
				[]byte{0xff, 0xfa, 0x18, 0, 0x56, 0x54, 0x32, 0x32, 0x30, 0xff, 0xf0},
			},
			want: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, tc, func() []byte {
				_ = tc.skipSBSequence()
				buf := make([]byte, tc.reader.Buffered())
				tc.reader.Read(buf)

				return buf
			})
		})
	}
}

func Test_skipCommand(t *testing.T) {
	tc := &TelnetClient{Timeout: 10 * time.Millisecond}
	tests := []testReadCase{
		{
			name: "skipCommand: Just skip SB/SE sequence",
			args: [][]byte{
				[]byte{0xfa, 0x18, 0, 0x56, 0x54, 0x32, 0x32, 0x30, 0xff, 0xf0},
			},
			want: []byte{},
		},
		{
			name: "skipCommand: Just skip DO command",
			args: [][]byte{
				[]byte{0xfd, 0x03},
			},
			want: []byte{},
		},
		{
			name: "skipCommand: Skip first DO command",
			args: [][]byte{
				[]byte{0xfd, 0x03, 0xff, 0xfd, 0x21},
			},
			want: []byte{0xff, 0xfd, 0x21},
		},
		{
			name: "skipCommand: Nothing, invalid format",
			args: [][]byte{
				[]byte{0xff, 0xfd, 0x03, 0xff, 0xfd, 0x21},
			},
			want: []byte{0xff, 0xfd, 0x03, 0xff, 0xfd, 0x21},
		},
		{
			name: "skipCommand: Plain text",
			args: [][]byte{
				[]byte{0x70, 0x6c, 0x61, 0x69, 0x6e},
			},
			want: []byte{0x70, 0x6c, 0x61, 0x69, 0x6e},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, tc, func() []byte {
				tc.skipCommand()
				buf := make([]byte, tc.reader.Buffered())
				tc.reader.Read(buf)

				return buf
			})
		})
	}
}

func Test_TelnetClient_ReadByte(t *testing.T) {
	tc := &TelnetClient{Timeout: 10 * time.Millisecond}
	tests := []testReadCase{
		{
			name: "ReadByte: Just byte",
			args: [][]byte{
				[]byte("a"),
			},
			want: []byte("a"),
		},
		{
			name:        "ReadByte: Empty",
			wantTimeout: true,
			args: [][]byte{
				[]byte{},
			},
			want: []byte{},
		},
		{
			name: "ReadByte: Byte with commands",
			args: [][]byte{
				[]byte{
					0xff, 0xfd, 0x03,
					0x70,
				},
			},
			want: []byte{0x70},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, tc, func() []byte {
				buf := make([]byte, 1)
				buf[0], _ = tc.ReadByte()
				return buf
			})
		})
	}
}

func Test_TelnetClient_ReadUntil(t *testing.T) {
	tc := &TelnetClient{Timeout: 10 * time.Millisecond}
	tests := []testReadCase{
		{
			name: "ReadUntil: One package",
			args: [][]byte{
				[]byte("one two"),
			},
			want: []byte("one "),
		},
		{
			name: "ReadUntil: two packages",
			args: [][]byte{
				[]byte("on"),
				[]byte("e two "),
			},
			want: []byte("one "),
		},
		{
			name:        "ReadUntil: timeout",
			wantTimeout: true,
			args: [][]byte{
				[]byte("one"),
			},
			want: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, tc, func() []byte {
				buf := make([]byte, 0, 1024)
				_, _ = tc.ReadUntil(&buf, ' ')
				return buf
			})
		})
	}
}

func Test_TelnetClient_ReadUntilPrompt(t *testing.T) {
	processor := func(data []byte) bool {
		return bytes.Compare(
			[]byte("Enter your username, please: "),
			data) == 0
	}

	tc := &TelnetClient{Timeout: 10 * time.Millisecond}
	tests := []testReadCase{
		{
			name: "ReadUntilPrompt: signle package",
			args: [][]byte{
				[]byte("Enter your username, please: "),
			},
			want: []byte("Enter your username, please: "),
		},
		{
			name: "ReadUntilPrompt: some packages",
			args: [][]byte{
				[]byte("Enter your u"),
				[]byte("sernam"),
				[]byte("e, please: "),
			},
			want: []byte("Enter your username, please: "),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, tc, func() []byte {
				buf, _ := tc.ReadUntilPrompt(processor)
				return buf
			})
		})
	}
}

func Test_TelnetClient_ReadUntilBanner(t *testing.T) {
	tc := &TelnetClient{Timeout: 10 * time.Millisecond}
	tests := []testReadCase{
		{
			name: "ReadUntilBanner: root banner",
			args: [][]byte{
				[]byte("1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue sta"),
				[]byte("te UNKNOWN mode DEFAULT group default qlen 1000\r\n"),
				[]byte("    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00\r\n"),
				[]byte("admin@RT-N14U:/tmp/home/root# "),
			},
			want: []byte(
				"1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue sta" +
					"te UNKNOWN mode DEFAULT group default qlen 1000\r\n" +
					"    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00\r\n",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, tc, func() []byte {
				buf, _ := tc.ReadUntilBanner()
				return buf
			})
		})
	}
}

func Test_TelnetClient_waitWelcomeSigns(t *testing.T) {
	tc := &TelnetClient{
		Timeout:  10 * time.Millisecond,
		Login:    "username",
		Password: "P@ssw0rd",
	}

	t.Run("waitWelcomeSigns", func(t *testing.T) {
		doneCh := make(chan bool)

		sr, sw := io.Pipe()
		cr, cw := io.Pipe()

		tc.reader = bufio.NewReader(cr)
		tc.writer = bufio.NewWriter(sw)

		// server
		go func() {
			login := make([]byte, 64)
			password := make([]byte, 64)

			// Send some stuff
			cw.Write([]byte{0xff, 0xfd, 0x03})
			cw.Write([]byte{
				0xff, 0xfb, 0x18,
				0xff, 0xfb, 0x1f,
				0xff, 0xfb, 0x20,
				0xff, 0xfb, 0x21,
				0xff, 0xfb, 0x22})

			// Login
			cw.Write([]byte("RT-N14U login: "))
			n, _ := sr.Read(login)
			if bytes.Compare(login[:n], []byte("username\r\n")) != 0 {
				t.Errorf(
					"waitWelcomeSigns: invalid login \"%s\"",
					login[:n])
				return
			}

			// Password
			cw.Write([]byte("\r\nPassword: "))
			n, _ = sr.Read(password)
			if bytes.Compare(password[:n], []byte("P@ssw0rd\r\n")) != 0 {
				t.Errorf(
					"waitWelcomeSigns: invalid password \"%s\"",
					password[:n])
				return
			}

			// Write banner
			cw.Write([]byte("\r\n\r\nASUSWRT RT-N14U_3.0.0.4 Sun Jan 19 14:13:45 UTC 2014"))
			cw.Write([]byte("admin@RT-N14U:/tmp/home/root# "))
		}()

		// client
		go func() {
			tc.waitWelcomeSigns()
			doneCh <- true
		}()

		select {
		case _ = <-doneCh:
			break
		case <-time.After(tc.Timeout):
			t.Errorf("waitWelcomeSigns: timeout is expired")
		}
	})
}
