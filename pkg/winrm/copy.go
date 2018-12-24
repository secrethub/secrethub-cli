package winrm

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sync"

	"bytes"

	"github.com/masterzen/winrm"
	"github.com/secrethub/secrethub-go/internals/api/uuid"
	"github.com/secrethub/secrethub-go/internals/errio"
)

var (
	maxOperationsPerShell = 15
)

// Inspired by https://github.com/packer-community/winrmcp
// The source was licensed under MIT.

// doCopy copies the contents from the io.Reader into the to path.
// The copy process is done in multiple steps:
//     1. The content is uploaded to a temporary file in chunks
//     2. The content is restored from the chunks into a single file at the target location.
//     3. The temporary file is removed.
//
// Progress is reported in the progress channel given as a variable.
func doCopy(client *winrm.Client, in io.Reader, toPath string, progress chan int) error {
	tempFile, err := tempFileName()
	if err != nil {
		return fmt.Errorf("error generating unique filename: %v", err)
	}
	tempPath := "$env:TEMP\\" + tempFile

	err = uploadContent(client, maxOperationsPerShell, "%TEMP%\\"+tempFile, in, progress)
	if err != nil {
		return fmt.Errorf("error uploading file to %s: %v", tempPath, err)
	}

	err = restoreContent(client, tempPath, toPath)
	if err != nil {
		return fmt.Errorf("error restoring file from %s to %s: %v", tempPath, toPath, err)
	}

	err = cleanupContent(client, tempPath)
	if err != nil {
		return fmt.Errorf("error removing temporary file %s: %v", tempPath, err)
	}

	return nil
}

// uploadContent uploads the contents in the io.Reader to the target location.
// The content is divided into multiple chunks.
// Each chunk is uploaded individually to the target location.
func uploadContent(client *winrm.Client, maxChunks int, filePath string, reader io.Reader, progress chan int) error {
	var err error
	done := false
	written := 0

	for !done {
		done, err = uploadChunks(client, filePath, maxChunks, reader)
		if err != nil {
			return err
		}

		if !done {
			written += chunkSize(filePath) * maxChunks
			progress <- written
		}
	}

	close(progress)

	return nil
}

// uploadChunks uploads the content by dividing the content into multiple chunks.
// The chunks are combined into a single file.
// The chunks are used to get around the maximum command line size limit.
// This allows us to use the winRM connection for uploading files.
func uploadChunks(client *winrm.Client, filePath string, maxChunks int, reader io.Reader) (done bool, err error) {
	shell, err := client.CreateShell()
	if err != nil {
		return false, fmt.Errorf("couldn't create shell: %v", err)
	}
	defer func() {
		errc := shell.Close()
		if errc != nil {
			// Err is returned, because a named return type is used.
			err = errc
		}
	}()

	// Upload the file in chunks to get around the Windows command line size limit.
	// Base64 encodes each set of three bytes into four bytes. In addition the output
	// is padded to always be a multiple of four.
	//
	//   ceil(n / 3) * 4 = m1 - m2
	//
	//   where:
	//     n  = bytes
	//     m1 = max (8192 character command limit.)
	//     m2 = len(filePath)

	chunkSize := chunkSize(filePath)
	chunk := make([]byte, chunkSize)

	if maxChunks == 0 {
		maxChunks = 1
	}

	for i := 0; i < maxChunks; i++ {
		n, err := reader.Read(chunk)

		if err != nil && err != io.EOF {
			return false, err
		}
		if n == 0 {
			return true, nil
		}

		content := base64.StdEncoding.EncodeToString(chunk[:n])
		if err = appendContent(shell, filePath, content); err != nil {
			return false, err
		}
	}

	return false, nil
}

// restoreContent restores the content at the target location using the a source location.
func restoreContent(client *winrm.Client, fromPath, toPath string) (err error) {
	shell, err := client.CreateShell()
	if err != nil {
		return err
	}

	defer func() {
		errc := shell.Close()
		if errc != nil {
			// Err is returned, because a named return type is used.
			err = errc
		}
	}()

	script := fmt.Sprintf(`
		$tmp_file_path = [System.IO.Path]::GetFullPath("%s")
		$dest_file_path = [System.IO.Path]::GetFullPath("%s".Trim("'"))
		if (Test-Path $dest_file_path) {
			rm $dest_file_path
		}
		else {
			$dest_dir = ([System.IO.Path]::GetDirectoryName($dest_file_path))
			New-Item -ItemType directory -Force -ErrorAction SilentlyContinue -Path $dest_dir | Out-Null
		}

		if (Test-Path $tmp_file_path) {
			$reader = [System.IO.File]::OpenText($tmp_file_path)
			$writer = [System.IO.File]::OpenWrite($dest_file_path)
			try {
				for(;;) {
					$base64_line = $reader.ReadLine()
					if ($base64_line -eq $null) { break }
					$bytes = [System.Convert]::FromBase64String($base64_line)
					$writer.write($bytes, 0, $bytes.Length)
				}
			}
			finally {
				$reader.Close()
				$writer.Close()
			}
		} else {
			echo $null > $dest_file_path
		}
	`, fromPath, toPath)

	cmd, err := shell.Execute(winrm.Powershell(script))
	if err != nil {
		return err
	}
	defer func() {
		errc := cmd.Close()
		if errc != nil {
			// Err is returned, because a named return type is used.
			err = errc
		}
	}()

	var wg sync.WaitGroup
	copyFunc := func(w io.Writer, r io.Reader, errChannel chan error) {
		defer wg.Done()
		_, err := io.Copy(w, r)
		errChannel <- err
	}

	wg.Add(1)
	var stdErr bytes.Buffer

	errChannel := make(chan error)
	go copyFunc(&stdErr, cmd.Stderr, errChannel)
	err = <-errChannel
	if err != nil {
		return errio.Error(err)
	}

	cmd.Wait()
	wg.Wait()

	if cmd.ExitCode() != 0 {
		return fmt.Errorf("restore operation returned code=%d", cmd.ExitCode())
	} else if stdErr.String() != "" {
		return fmt.Errorf("restore operation returned error: %s", stdErr.String())
	}

	return nil
}

// cleanupContent removes a temporary file at the filePath location.
func cleanupContent(client *winrm.Client, filePath string) (err error) {
	shell, err := client.CreateShell()
	if err != nil {
		return err
	}

	defer func() {
		errc := shell.Close()
		if errc != nil {
			// Err is returned, because a named return type is used.
			err = errc
		}
	}()
	cmd, err := shell.Execute("powershell", "Remove-Item", filePath, "-ErrorAction SilentlyContinue")
	if err != nil {
		return err
	}

	cmd.Wait()
	defer func() {
		errc := cmd.Close()
		if errc != nil {
			// Err is returned, because a named return type is used.
			err = errc
		}
	}()
	return nil
}

// appendContent appends content to the temporary file.
func appendContent(shell *winrm.Shell, filePath, content string) (err error) {
	cmd, err := shell.Execute(fmt.Sprintf("echo %s >> \"%s\"", content, filePath))

	if err != nil {
		return err
	}

	defer func() {
		errc := cmd.Close()
		if errc != nil {
			// Err is returned, because a named return type is used.
			err = errc
		}
	}()
	var wg sync.WaitGroup
	copyFunc := func(w io.Writer, r io.Reader, errChannel chan error) {
		defer wg.Done()
		_, errc := io.Copy(w, r)
		errChannel <- errc
	}

	wg.Add(2)
	errChannel := make(chan error)
	go copyFunc(os.Stdout, cmd.Stdout, errChannel)
	go copyFunc(os.Stderr, cmd.Stderr, errChannel)

	err2, err1 := <-errChannel, <-errChannel
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	cmd.Wait()
	wg.Wait()

	if cmd.ExitCode() != 0 {
		return fmt.Errorf("upload operation returned code=%d", cmd.ExitCode())
	}

	return nil
}

func tempFileName() (string, error) {
	return fmt.Sprintf("winrmcp-%s.tmp", uuid.New()), nil
}

func chunkSize(filePath string) int {
	return ((8000 - len(filePath)) / 4) * 3
}
