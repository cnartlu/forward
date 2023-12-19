package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

type AsnIpinfo struct {
	Asn    string `maxminddb:"asn"`
	Domain string `maxminddb:"domain"`
	Name   string `maxminddb:"name"`
}

type mmdb struct {
	path   string
	dlUrl  string
	reader *maxminddb.Reader
	rw     sync.RWMutex
}

func (d *mmdb) SetPath(s string) {
	if s == "" {
		s = "asn.mmdb"
	}
	if !filepath.IsAbs(s) {
		execDir, _ := os.Executable()
		execDir, _ = filepath.EvalSymlinks(execDir)
		s = filepath.Join(filepath.Dir(execDir), s)
	}
	d.path = s
}

func (d *mmdb) Getpath() string {
	return d.path
}

func (d *mmdb) open() error {
	reader, err := maxminddb.Open(d.Getpath())
	if err != nil {
		return err
	}
	if err := d.close(); err != nil {
		log.Default().Printf("close mmdb %s\n", err)
	}
	d.reader = reader
	return nil
}

func (d *mmdb) Open() error {
	d.rw.Lock()
	err := d.open()
	d.rw.Unlock()
	return err
}

func (d *mmdb) Reader() *maxminddb.Reader {
	d.rw.RLock()
	if d.reader == nil {
		if err := d.open(); err != nil {
			d.rw.RUnlock()
			panic(err)
		}
	}
	reader := d.reader
	d.rw.RUnlock()
	return reader
}

func (d *mmdb) close() error {
	if d != nil && d.reader != nil {
		reader := d.reader
		d.reader = nil
		return reader.Close()
	}
	return nil
}

func (d *mmdb) Close() error {
	d.rw.Lock()
	err := d.close()
	d.rw.Unlock()
	return err
}

func (d *mmdb) DownLoad() error {
	req := http.Client{}
	res, err := req.Get(d.dlUrl)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		s, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("%s,io %w", s, err)
		}
		return fmt.Errorf("%s", s)
	}
	filename := d.Getpath()
	_, err = os.Stat(filename)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("mmdb %s: %w", filename, err)
	}
	d.rw.Lock()
	defer d.rw.Unlock()
	if err == nil {
		if err := os.Rename(filename, filename+".bak"); err != nil {
			return fmt.Errorf("rename used %s: %w", filename, err)
		}
	}
	if err := func() error {
		f, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("create file %w", err)
		}
		if _, err := io.Copy(f, res.Body); err != nil {
			return fmt.Errorf("copy file %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close file %w", err)
		}
		_ = os.Remove(filename + ".bak")
		return nil
	}(); err != nil {
		_ = os.Rename(filename+".bak", filename)
		return err
	}
	d.close()
	return nil
}

func InitMMDB(url, path string) *mmdb {
	db := mmdb{dlUrl: url}
	db.SetPath(path)
	filename := db.Getpath()
	fi, err := os.Stat(filename)
	if os.IsNotExist(err) {
		if err := db.DownLoad(); err != nil {
			log.Default().Fatalf("download domain asn: %s", err)
		}
		return &db
	}
	if fi.IsDir() {
		log.Default().Fatalf("%s is directory, that must is file", filename)
	}
	return &db
}
