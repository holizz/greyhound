package greyhound

import (
	"bufio"
	"errors"
	"io"
	"os/exec"
	"time"
)

// https://gist.github.com/tamagokun/3801087
var routerPhp = `<?php
 
$root = $_SERVER['DOCUMENT_ROOT'];
chdir($root);
$path = '/'.ltrim(parse_url($_SERVER['REQUEST_URI'])['path'],'/');
set_include_path(get_include_path().':'.__DIR__);
if(file_exists($root.$path))
{
	if(is_dir($root.$path) && substr($path,strlen($path) - 1, 1) !== '/')
		$path = rtrim($path,'/').'/index.php';
	if(strpos($path,'.php') === false) return false;
	else {
		chdir(dirname($root.$path));
		require_once $root.$path;
	}
}else include_once 'index.php';
`

// A low-level command
// Starts PHP running, waits half a second, returns an error if PHP stopped during that time
func runPhp(dir, host string, extraArgs []string) (cmd *exec.Cmd, stdout chan string, stderr chan string, errorChan chan error, err error) {
	args := []string{
		"-n", // do not read php.ini
		"-S", host,
		"-t", dir,
		"-d", "display_errors=Off",
		"-d", "log_errors=On",
		"-d", "error_reporting=E_ALL",
		"-d", "upload_max_filesize=1024G",
		"-d", "post_max_size=1024G",
		"-d", "max_execution_time=0",
	}
	args = append(args, extraArgs...)

	cmd = exec.Command("php", args...)

	// Connect stdin
	_stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	// Connect stdout
	_stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	stdout = chanify(&_stdout)

	// Connect stderr
	_stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}
	stderr = chanify(&_stderr)

	// Let's go
	errorChan = make(chan error)

	go func() {
		err = cmd.Run()
		defer _stdin.Close()
		io.WriteString(_stdin, routerPhp)
		if err != nil {
			errorChan <- err
		} else {
			errorChan <- errors.New("command exited early")
		}
	}()

	// Wait 1 second for the command to terminate
	// If it exits early, that's bad whatever the exit status

	select {
	case <-time.After(time.Second):
		return
	case err = <-errorChan:
		return
	}
}

func chanify(pipe *io.ReadCloser) (ch chan string) {
	ch = make(chan string)
	scanner := bufio.NewScanner(*pipe)

	go func() {
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		err := scanner.Err()
		if err != nil {
			panic(err)
		}
	}()

	return
}
