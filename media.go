package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ryanss/gorm"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ValidFileType = map[string]bool{
	".m4a": true,
	".m4v": true,
	".mp3": true,
	".mp4": true,
}

type Media struct {
	Id           int
	Type         string
	Path         string
	Filename     string
	Size         int64
	Title        string
	Desc         string
	Runtime      int
	Genres       string
	Year         int
	Poster       string
	Season       int
	Episode      int
	EpisodeTitle string
	EpisodeDesc  string
	EpisodeAired time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    time.Time
}

func (m Media) TableName() string {
	return "media"
}

func (m Media) PubDate() string {
	return m.CreatedAt.Format(time.RFC1123)
}

func (m Media) MediaURL(host string) string {
	Url, _ := url.Parse(fmt.Sprintf("http://%s/%s/%d/%s", host, m.Type, m.Id, m.Filename))
	return Url.String()
}

func ProcessFile(fp string, timestamp time.Time) {
	fmt.Println(fp)
	file, _ := os.Stat(fp)
	fmt.Println(file.Name())
	media := Media{
		Path:     fp,
		Filename: file.Name(),
	}
	db.Where(media).FirstOrCreate(&media)
	media.Size = file.Size()

	if media.Type != "" {
		db.Save(&media)
		return
	}

	// Audio
	if filepath.Ext(file.Name()) == ".mp3" {
		media.Type = "audio"
	}

	// TV Show
	re := regexp.MustCompile("S[0-9]{2}E[0-9]{2}")
	info := re.FindString(media.Filename)
	if info != "" {
		media.Type = "tvshow"
		media.Season, _ = strconv.Atoi(info[1:3])
		media.Episode, _ = strconv.Atoi(info[4:])
		media.ScrapeTVShow()
	}

	// Movie
	if media.Type == "" {
		filename := []byte(media.Filename)
		index := len(filename)
		reYear := regexp.MustCompile("\\.[0-9]{4}")
		iYear := reYear.FindIndex(filename)
		if iYear != nil && iYear[0] < index {
			index = iYear[0]
		}
		reExt := regexp.MustCompile("\\.[a-z0-9]+$")
		iExt := reExt.FindIndex(filename)
		if iExt != nil && iExt[0] < index {
			index = iExt[0]
		}
		media.Title = strings.Replace(string(filename[0:index]), ".", " ", -1)
		if iYear != nil {
			year, _ := strconv.ParseInt(string(filename[iYear[0]+1:iYear[1]]), 10, 0)
			media.Year = int(year)
		}
		media.ScrapeMovie()
		if media.Desc != "" {
			media.Type = "movie"
		}
	}

	// Video
	if media.Type == "" {
		media.Type = "video"
	}

	db.Save(&media)
}

func (m *Media) ScrapeMovie() {
	searchURL := "https://www.themoviedb.org/search?query=" + m.Title
	searchURL = strings.Replace(searchURL, " ", "%20", -1)
	doc, _ := goquery.NewDocument(searchURL)
	s := doc.Find("ul.movie li").First()
	s = s.Find("a").First()
	link, _ := s.Attr("href")
	doc, _ = goquery.NewDocument("https://www.themoviedb.org" + link)
	s = doc.Find("#overview").First()
	m.Desc = s.Text()
	doc.Find("#genres span").Each(func(i int, s *goquery.Selection) {
		m.Genres = m.Genres + s.Text() + ", "
	})
	if m.Genres != "" {
		m.Genres = m.Genres[:len(m.Genres)-2]
	}
	s = doc.Find("a.poster").First()
	m.Poster, _ = s.Find("img").Attr("src")
	runtime, _ := strconv.ParseInt(doc.Find("#runtime").Text(), 10, 0)
	m.Runtime = int(runtime)
}

func (m *Media) ScrapeTVShow() {
}

func initDB() gorm.DB {
	db, err := gorm.Open("sqlite3", config.Database)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	db.DB()
	db.LogMode(true)
	db.AutoMigrate(&Media{})
	return db
}

var db = initDB()

func updateDB() {
	timestamp := time.Now().Local()

	for _, dir := range config.Media {
		d, _ := os.Open(dir)
		defer d.Close()
		files, _ := d.Readdir(-1)
		for _, file := range files {
			if ValidFileType[filepath.Ext(file.Name())] {
				ProcessFile(dir+string(filepath.Separator)+file.Name(), timestamp)
			}

		}
	}

	// Soft delete records that were not found
	db.Where("updated_at < ?", timestamp).Delete(Media{})
}