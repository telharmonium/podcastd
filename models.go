package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

type Movie struct {
	Id        int
	Filename  string
	Size      int64
	Title     string
	Year      int
	Added     time.Time
	Timestamp time.Time
}

func (m Movie) PubDate() string {
	return m.Added.Format(time.RFC1123)
}

type TVShow struct {
	Id           int
	Filename     string
	Size         int64
	ShowTitle    string
	Season       int
	Episode      int
	EpisodeTitle string
	EpisodeDesc  string
	Aired        time.Time
	Timestamp    time.Time
}

func (t TVShow) TableName() string {
	return "tvshows"
}

func (t *TVShow) Parse() {
	re := regexp.MustCompile("S[0-9]{2}E[0-9]{2}")
	fmt.Println(t.Filename)
	info := re.FindString(t.Filename)
	fmt.Println(info)
	t.Season, _ = strconv.Atoi(info[1:3])
	fmt.Println(t.Season)
	t.Episode, _ = strconv.Atoi(info[5:])
	fmt.Println(t.Episode)
}

func initDB() gorm.DB {
	db, err := gorm.Open("sqlite3", dir+"btpodcast.db")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	db.DB()
	db.LogMode(true)
	db.AutoMigrate(&Movie{}, &TVShow{})
	return db
}

var db = initDB()

func updateDB() {
	timestamp := time.Now().Local()

	d, _ := os.Open(dir + "movies")
	defer d.Close()
	files, _ := d.Readdir(-1)
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".m4v" {
			movie := Movie{}
			db.Where(Movie{
				Filename: file.Name(),
				Size:     file.Size(),
			}).Assign(Movie{Timestamp: timestamp}).FirstOrCreate(&movie)
		}
	}
	db.Exec("UPDATE movies SET added=datetime(?, 'localtime') WHERE added < '1990-01-01';", timestamp)

	d, _ = os.Open(dir + "tvshows")
	defer d.Close()
	files, _ = d.Readdir(-1)
	for _, file := range files {
		if file.IsDir() {
			e, _ := os.Open(dir + "tvshows" + string(filepath.Separator) + file.Name())
			defer e.Close()
			episodes, _ := e.Readdir(-1)
			for _, episode := range episodes {
				if filepath.Ext(episode.Name()) == ".m4v" {
					show := TVShow{
						ShowTitle: file.Name(),
						Filename:  episode.Name(),
						Size:      episode.Size(),
					}
					show.Parse()
					db.Where(show).Assign(TVShow{Timestamp: timestamp}).FirstOrCreate(&show)
				}
			}
		}
	}

	// Remove records from database that were not found
	db.Where("timestamp <> ?", timestamp).Delete(Movie{})
	db.Where("timestamp <> ?", timestamp).Delete(TVShow{})
}
