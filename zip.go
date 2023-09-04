package myutil

import (
	"archive/zip"
	"compress/flate"
	"fmt"
	ezip "github.com/alexmullins/zip"
	"github.com/mholt/archiver"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// archive
func archive(dir, zipFilename string) error {
	z := archiver.Zip{
		CompressionLevel:       flate.DefaultCompression,
		MkdirAll:               true,
		SelectiveCompression:   true,
		ContinueOnError:        false,
		OverwriteExisting:      true,
		ImplicitTopLevelFolder: false,
	}
	return z.Archive([]string{dir}, zipFilename)
}

// CompressUsing7z Compress using 7z
func CompressUsing7z(src, dst, password, exePath string) ([]byte, error) {
	if exePath == "" {
		sysType := runtime.GOOS
		if sysType == "windows" {
			exePath = "./7z/7z.exe"
		} else if sysType == "darwin" {
			exePath = "/opt/homebrew/bin/7z"
		}
	}
	if !FileExists(exePath) {
		return nil, fmt.Errorf("找不到%v, 请将7z执行文件放置到相应目录", exePath)
	}

	args := []string{"a", dst, src}
	if password != "" {
		args = append(args, "-p"+password)
	}
	cmd := exec.Command(exePath, args...)
	return cmd.CombinedOutput()
}

// Compress 压缩文件夹
func Compress(path, targetFile, password string) error {
	d, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer d.Close()
	w := ezip.NewWriter(d)
	defer w.Close()

	f, err := os.Open(path)
	if err != nil {
		return err
	}

	err = comp(f, "", w, password)

	return err
}

// EncryptZip 加密压缩文件
func CompressWithPassword(src, dst, password string) error {
	zipfile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := ezip.NewWriter(zipfile)
	defer archive.Close()

	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if src == path {
			return nil
		}
		header, err := ezip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(path, filepath.Dir(src)+"/")
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		// 设置密码
		header.SetPassword(password)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
		}
		return err
	})
	return err
}

func comp(file *os.File, prefix string, zw *ezip.Writer, password string) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			err = comp(f, prefix, zw, password)
			if err != nil {
				return err
			}
		}
	} else {
		header, err := ezip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = prefix + "/" + header.Name
		if password != "" {
			header.SetPassword(password)
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func UnCompress(dst, src string) (dir_name string, err error) {
	// 打开压缩文件，这个 zip 包有个方便的 ReadCloser 类型
	// 这个里面有个方便的 OpenReader 函数，可以比 tar 的时候省去一个打开文件的步骤
	zr, err := zip.OpenReader(src)
	defer zr.Close()
	if err != nil {
		return dir_name, err
	}

	// 如果解压后不是放在当前目录就按照保存目录去创建目录
	if dst != "" {
		if err := os.MkdirAll(dst, 0755); err != nil {
			return dir_name, err
		}
	}

	// 遍历 zr ，将文件写入到磁盘
	for _, file := range zr.File {
		path := filepath.Join(dst, file.Name)

		// 如果是目录，就创建目录
		if file.FileInfo().IsDir() {
			os.RemoveAll(path)
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return dir_name, err
			}
			dir_name = file.Name[0 : len(file.Name)-1]
			// 因为是目录，跳过当前循环，因为后面都是文件的处理
			continue
		}

		// 获取到 Reader
		fr, err := file.Open()
		if err != nil {
			return dir_name, err
		}

		// 创建要写出的文件对应的 Write
		fw, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, file.Mode())
		if err != nil {
			return dir_name, err
		}

		_, err = io.Copy(fw, fr)
		if err != nil {
			return dir_name, err
		}

		// 将解压的结果输出
		//fmt.Printf("成功解压 %s\n", path)

		// 因为是在循环中，无法使用 defer ，直接放在最后
		// 不过这样也有问题，当出现 err 的时候就不会执行这个了，
		// 可以把它单独放在一个函数中，这里是个实验，就这样了
		fw.Close()
		fr.Close()
	}
	return dir_name, nil
}
