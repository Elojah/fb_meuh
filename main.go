/* ************************************************************************** */
/*                                                                            */
/*                                                        :::      ::::::::   */
/*   main.go                                            :+:      :+:    :+:   */
/*                                                    +:+ +:+         +:+     */
/*   By: hdezier <hdezier@student.42.fr>            +#+  +:+       +#+        */
/*                                                +#+#+#+#+#+   +#+           */
/*   Created: 2017/03/04 14:11:08 by hdezier           #+#    #+#             */
/*   Updated: 2017/03/04 23:24:23 by hdezier          ###   ########.fr       */
/*                                                                            */
/* ************************************************************************** */

package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/camelcase"
	fb "github.com/huandu/facebook"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

func clean_song_title(title string) (clean_song_title string) {
	// Ignore album here ?
	r := regexp.MustCompile(`\([^)]*\)`)
	clean_song_title = ``
	title = strings.Trim(title[4:], `. `)
	title = string(r.ReplaceAll([]byte(title), []byte(``)))
	title = strings.Replace(title, `-`, ` `, -1)
	song_title_splitted := camelcase.Split(title)
	for key := range song_title_splitted {
		clean_val := strings.Trim(song_title_splitted[key], `. `)
		if len(clean_val) > 0 {
			clean_song_title += ` ` + clean_val
		}
	}
	return
}

func search_vgoogle_best_result(song_title string) (matches []string) {
	vggl_search_url := `https://www.google.fr/search?`
	cleaned_song_title := clean_song_title(song_title)
	query_params := map[string]string{
		`q`:   url.QueryEscape(cleaned_song_title),
		`tbm`: `vid`,
	}
	for key, val := range query_params {
		vggl_search_url += key + `=` + val + `&`
	}
	vggl_search_url = vggl_search_url[:len(vggl_search_url)-1]
	fmt.Println(vggl_search_url)
	doc, err := goquery.NewDocument(vggl_search_url)
	if err != nil {
		fmt.Println(`Google search failed :(`)
		fmt.Println(err.Error())
		return
	}
	best_ortho := doc.Find(`a.spell_orig`).First()
	if best_ortho != nil {
		href, exists := best_ortho.Attr(`href`)
		if exists {
			fmt.Println(`Google found better query:` + href)
			doc, err = goquery.NewDocument(href)
			if err != nil {
				fmt.Println(`Google search failed :(`)
				fmt.Println(err.Error())
				return
			}
		}
	}
	doc.Find(`div.g`).Each(func(i int, s *goquery.Selection) {
		a_link := s.Find(`h3.r`).First().Children()
		href, exists := a_link.Attr(`href`)
		if exists {
			spec_ggl_idx := strings.Index(href, `&sa`)
			clean_url := href[7:spec_ggl_idx]
			clean_url = strings.Replace(clean_url, `%3Fv%3D`, `?v=`, -1)
			matches = append(matches, clean_url)
		}
	})
	return
}

func find_current_song_title(doc *goquery.Document) string {
	return doc.Find(`table`).Children().First().Children().First().Text()
}

func post_results_to_fb(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	playlist_url := `http://www.radiomeuh.com/playlist/`
	doc, err := goquery.NewDocument(playlist_url)
	if err != nil {
		fmt.Println(`Playlist link is dead :(`)
		return
	}
	song_title := find_current_song_title(doc)
	video_urls := search_vgoogle_best_result(song_title)
	fmt.Println(video_urls)

	r.ParseForm()
	code := r.FormValue(`code`)

	token, err := oauthConf.Exchange(oauth2.NoContext, code)

	// Create a client to manage access token life cycle.
	client := oauthConf.Client(oauth2.NoContext, token)

	// Use OAuth2 client with session.
	session := &fb.Session{
		Version:    "v2.4",
		HttpClient: client,
	}

	// Use session.
	res, err := session.Post(`/me/feed`, fb.Params{
		`message`: video_urls[0],
	})
	fmt.Println(res, err)
	w.Write([]byte(`OK :)`))
}

func main() {
	r := httprouter.New()
	r.ServeFiles("/static/*filepath", http.Dir("static"))

	r.GET("/", handleMain)
	r.GET("/login", handleFacebookLogin)
	r.GET("/oauth2callback", handleFacebookCallback)
	r.GET(`/api/post_music_fb`, post_results_to_fb)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Server listening on port: " + port)
	http.ListenAndServe(":"+port, r)
}
