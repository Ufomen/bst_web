package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chris-sg/bst_server_models"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var (
	bstApiClient *http.Client
)

func InitClient() {
	bstApiClient = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar:           nil,
		Timeout:       time.Second * 45,
	}
}

// CreateBstApiRouter will generate a router mapped against BST API. Middleware
// may be passed in to then be used by certain routes.
func CreateBstApiRouter(prefix string, middleware map[string]*negroni.Negroni) *mux.Router {
	bstApiRouter := mux.NewRouter().PathPrefix(prefix + "/bst_api").Subrouter()
	bstApiRouter.Path("/status").Handler(negroni.New(
		negroni.Wrap(http.HandlerFunc(StatusGet)))).Methods(http.MethodGet)
	bstApiRouter.Path("/eagate_login").Handler(negroni.New(
		negroni.Wrap(http.HandlerFunc(EagateLoginPost)))).Methods(http.MethodPost)
	bstApiRouter.Path("/eagate_logout").Handler(negroni.New(
		negroni.Wrap(http.HandlerFunc(EagateLogoutPost)))).Methods(http.MethodPost)

	bstApiRouter.Path("/ddr_update").Handler(negroni.New(
		negroni.Wrap(http.HandlerFunc(DdrUpdatePatch)))).Methods(http.MethodPatch)
	bstApiRouter.Path("/ddr_refresh").Handler(negroni.New(
		negroni.Wrap(http.HandlerFunc(DdrRefreshPatch)))).Methods(http.MethodPatch)
	bstApiRouter.Path("/ddr_stats").Handler(negroni.New(
		negroni.Wrap(http.HandlerFunc(DdrStatsGet)))).Methods(http.MethodGet)


	return bstApiRouter
}

// StatusGet will call StatusGetImpl() and return the result.
func StatusGet(rw http.ResponseWriter, r *http.Request) {
	status := StatusGetImpl()

	bytes, _ := json.Marshal(status)
	rw.WriteHeader(http.StatusOK)
	rw.Write(bytes)
}

// StatusGetImpl will retrieve the current state of the api, the database and eagate.
func StatusGetImpl() (status bst_models.ApiStatus) {
	uri, _ := url.Parse("https://" + bstApi + bstApiBase + "status")

	status.Api = "bad"
	status.EaGate = "bad"
	status.Db = "bad"

	req := &http.Request{
		Method:           http.MethodGet,
		URL:              uri,
	}
	res, err := bstApiClient.Do(req)
	if err != nil {
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	err = json.Unmarshal(body, &status)
	if err != nil {
		status.Api = "unknown"
	}

	return
}

func EagateLoginGet(rw http.ResponseWriter, r *http.Request) {
	token, err := TokenForRequest(r)
	if err != nil {
		status := bst_models.Status{
			Status:  "bad",
			Message: err.Error(),
		}

		bytes, _ := json.Marshal(status)
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write(bytes)
		return
	}
	status, users := EagateLoginGetImpl(token)

	if status.Status == "bad" {
		bytes, _ := json.Marshal(status)
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write(bytes)
		return
	}

	bytes, _ := json.Marshal(users)
	rw.WriteHeader(http.StatusOK)
	rw.Write(bytes)
	return
}

func EagateLoginGetImpl(token string) (status bst_models.Status, users []bst_models.EagateUser){

	uri, _ := url.Parse("https://" + bstApi + bstApiBase + "user/login")

	req := &http.Request{
		Method:           http.MethodGet,
		URL:              uri,
		Header:			  make(map[string][]string),
	}
	req.Header.Add("Authorization", "Bearer " + token)

	res, err := bstApiClient.Do(req)
	if err != nil {
		status.Status = "bad"
		status.Message = "api error"
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	users = make([]bst_models.EagateUser, 0)
	json.Unmarshal(body, &users)

	status.Status = "ok"
	status.Message = fmt.Sprintf("found %d users", len(users))
	return
}

func EagateLoginPost(rw http.ResponseWriter, r *http.Request) {
	token, err := TokenForRequest(r)
	if err != nil {
		status := bst_models.Status{
			Status:  "bad",
			Message: err.Error(),
		}

		bytes, _ := json.Marshal(status)
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write(bytes)
		return
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		status := bst_models.Status{
			Status:  "bad",
			Message: err.Error(),
		}

		bytes, _ := json.Marshal(status)
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write(bytes)
		return
	}

	loginRequest := bst_models.LoginRequest{}
	json.Unmarshal(body, &loginRequest)

	status := EagateLoginPostImpl(token, loginRequest)

	bytes, _ := json.Marshal(status)
	if status.Status == "ok" {
		rw.WriteHeader(http.StatusOK)
	} else {
		rw.WriteHeader(http.StatusInternalServerError)
	}
	rw.Write(bytes)
	return
}

func EagateLoginPostImpl(token string, loginRequest bst_models.LoginRequest) (status bst_models.Status) {
	uri, _ := url.Parse("https://" + bstApi + bstApiBase + "user/login")

	req := &http.Request{
		Method:           http.MethodPost,
		URL:              uri,
		Header:			  make(map[string][]string),
	}
	req.Header.Add("Authorization", "Bearer " + token)

	b, _ := json.Marshal(loginRequest)
	req.Body = ioutil.NopCloser(bytes.NewReader(b))

	res, err := bstApiClient.Do(req)
	if err != nil {
		status.Status = "bad"
		status.Message = "api error"
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	json.Unmarshal(body, &status)

	return
}

func EagateLogoutPost(rw http.ResponseWriter, r *http.Request) {
	token, err := TokenForRequest(r)
	if err != nil {
		status := bst_models.Status{
			Status:  "bad",
			Message: err.Error(),
		}

		bytes, _ := json.Marshal(status)
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write(bytes)
		return
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		status := bst_models.Status{
			Status:  "bad",
			Message: err.Error(),
		}

		bytes, _ := json.Marshal(status)
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write(bytes)
		return
	}

	logoutRequest := bst_models.LogoutRequest{}
	err = json.Unmarshal(body, &logoutRequest)
	fmt.Println(err)
	fmt.Printf("%s\n", body)
	fmt.Println(logoutRequest)

	status := EagateLogoutPostImpl(token, logoutRequest)

	bytes, _ := json.Marshal(status)
	if status.Status == "ok" {
		rw.WriteHeader(http.StatusOK)
	} else {
		fmt.Printf("failed to logout user: %s\n", status.Message)
		rw.WriteHeader(http.StatusInternalServerError)
	}
	rw.Write(bytes)
	return
}

func EagateLogoutPostImpl(token string, logoutRequest bst_models.LogoutRequest) (status bst_models.Status) {
	uri, _ := url.Parse("https://" + bstApi + bstApiBase + "user/logout")

	req := &http.Request{
		Method:           http.MethodPost,
		URL:              uri,
		Header:			  make(map[string][]string),
	}
	req.Header.Add("Authorization", "Bearer " + token)

	b, _ := json.Marshal(logoutRequest)
	req.Body = ioutil.NopCloser(bytes.NewReader(b))

	res, err := bstApiClient.Do(req)
	if err != nil {
		status.Status = "bad"
		status.Message = "api error"
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	json.Unmarshal(body, &status)

	return
}

func DdrUpdatePatch(rw http.ResponseWriter, r *http.Request) {
	token, err := TokenForRequest(r)
	if err != nil {
		status := bst_models.Status{
			Status:  "bad",
			Message: err.Error(),
		}

		bytes, _ := json.Marshal(status)
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write(bytes)
		return
	}

	status := DdrUpdatePatchImpl(token)

	bytes, _ := json.Marshal(status)
	if status.Status == "ok" {
		rw.WriteHeader(http.StatusOK)
	} else {
		fmt.Printf("failed to update ddr profile: %s\n", status.Message)
		rw.WriteHeader(http.StatusInternalServerError)
	}
	rw.Write(bytes)
	return
}

func DdrUpdatePatchImpl(token string) (status bst_models.Status) {
	uri, _ := url.Parse("https://" + bstApi + bstApiBase + "ddr/profile/update")

	req := &http.Request{
		Method:           http.MethodPatch,
		URL:              uri,
		Header:			  make(map[string][]string),
	}
	req.Header.Add("Authorization", "Bearer " + token)

	res, err := bstApiClient.Do(req)
	if err != nil {
		status.Status = "bad"
		status.Message = "api error"
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	json.Unmarshal(body, &status)

	return
}

func DdrRefreshPatch(rw http.ResponseWriter, r *http.Request) {
	token, err := TokenForRequest(r)
	if err != nil {
		status := bst_models.Status{
			Status:  "bad",
			Message: err.Error(),
		}

		bytes, _ := json.Marshal(status)
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write(bytes)
		return
	}

	status := DdrRefreshPatchImpl(token)

	bytes, _ := json.Marshal(status)
	if status.Status == "ok" {
		rw.WriteHeader(http.StatusOK)
	} else {
		fmt.Printf("failed to refresh ddr profile: %s\n", status.Message)
		rw.WriteHeader(http.StatusInternalServerError)
	}
	rw.Write(bytes)
	return
}

func DdrRefreshPatchImpl(token string) (status bst_models.Status) {
	uri, _ := url.Parse("https://" + bstApi + bstApiBase + "ddr/profile/refresh")

	req := &http.Request{
		Method:           http.MethodPatch,
		URL:              uri,
		Header:			  make(map[string][]string),
	}
	req.Header.Add("Authorization", "Bearer " + token)

	res, err := bstApiClient.Do(req)
	if err != nil {
		status.Status = "bad"
		status.Message = "api error"
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	json.Unmarshal(body, &status)

	return
}

func DdrStatsGet(rw http.ResponseWriter, r *http.Request) {
	token, err := TokenForRequest(r)
	if err != nil {
		status := bst_models.Status{
			Status:  "bad",
			Message: err.Error(),
		}

		bytes, _ := json.Marshal(status)
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write(bytes)
		return
	}

	stats := DdrStatsGetImpl(token)

	rw.Write([]byte(stats))
	return
}

func DdrStatsGetImpl(token string) (stats string) {
	uri, _ := url.Parse("https://" + bstApi + bstApiBase + "ddr/songs/scores/extended")

	req := &http.Request{
		Method:           http.MethodGet,
		URL:              uri,
		Header:			  make(map[string][]string),
	}
	req.Header.Add("Authorization", "Bearer " + token)

	res, err := bstApiClient.Do(req)
	if err != nil {
		stats = "<a>API Error</a>"
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	statsFromServer := make([]bst_models.DdrStatisticsTable, 0)

	err = json.Unmarshal(body, &statsFromServer)
	if err != nil {
		stats = "<a>API Error</a>"
		return
	}

stats = `<button class="btn btn-primary" type="button" data-toggle="collapse" data-target="#collapseFilter" aria-expanded="false" aria-controls="collapseFilter">
    Filtering
  </button>
</p>
<div class="collapse" id="collapseFilter">
  <div class="card card-body">
	<div class="container">
		<div class="row">
			<div class="container">
				<div class="row">Levels</div>
				<div class="row">
					<div class="col enabled level-filter" id="level-filter-1">1</div>
					<div class="col enabled level-filter" id="level-filter-2">2</div>
					<div class="col enabled level-filter" id="level-filter-3">3</div>
					<div class="col enabled level-filter" id="level-filter-4">4</div>
					<div class="col enabled level-filter" id="level-filter-5">5</div>
				</div>
				<div class="row">
					<div class="col enabled level-filter" id="level-filter-6">6</div>
					<div class="col enabled level-filter" id="level-filter-7">7</div>
					<div class="col enabled level-filter" id="level-filter-8">8</div>
					<div class="col enabled level-filter" id="level-filter-9">9</div>
					<div class="col enabled level-filter" id="level-filter-10">10</div>
				</div>
				<div class="row">
					<div class="col enabled level-filter" id="level-filter-11">11</div>
					<div class="col enabled level-filter" id="level-filter-12">12</div>
					<div class="col enabled level-filter" id="level-filter-13">13</div>
					<div class="col enabled level-filter" id="level-filter-14">14</div>
					<div class="col enabled level-filter" id="level-filter-15">15</div>
				</div>
				<div class="row">
					<div class="col enabled level-filter" id="level-filter-16">16</div>
					<div class="col enabled level-filter" id="level-filter-17">17</div>
					<div class="col enabled level-filter" id="level-filter-18">18</div>
					<div class="col enabled level-filter" id="level-filter-19">19</div>
					<div class="col"></div>
				</div>
				<div class="row">
					<div class="col"></div>
					<div class="col" id="level-filter-all-enable">All</div>
					<div class="col"></div>
					<div class="col" id="level-filter-all-disable">None</div>
					<div class="col"></div>
				</div>
			</div>
		</div>
		<div class="row">
		</div>
		<div class="row">
		</div>
		<div class="row">
		</div>
		<div class="row">
		</div>
	</div>
	<table border="0" cellspacing="5" cellpadding="5">
		<thead>
			<tr>
				<th>Mode</th>
				<th>Difficulty</th>
				<th>Lamp</th>
			</tr>
		</thead>
        <tbody><tr>
            <td><input type="checkbox" id="single-filter" name="single-filter" checked> SINGLE</td>
			<td><input type="checkbox" id="beginner-filter" name="beginner-filter" checked> BEGINNER</td>
			<td><input type="checkbox" id="fail-filter" name="fail-filter" checked> FAIL</td>
        </tr><tr>
            <td><input type="checkbox" id="double-filter" name="double-filter" checked> DOUBLE</td>
			<td><input type="checkbox" id="basic-filter" name="basic-filter" checked> BASIC</td>
			<td><input type="checkbox" id="clear-filter" name="clear-filter" checked> CLEAR</td>
        </tr><tr>
            <td></td>
			<td><input type="checkbox" id="difficult-filter" name="difficult-filter" checked> DIFFICULT </td>
			<td><input type="checkbox" id="good-filter" name="good-filter" checked> GOOD FC</td>
        </tr><tr>
            <td></td>
			<td><input type="checkbox" id="expert-filter" name="expert-filter" checked> EXPERT</td>
			<td><input type="checkbox" id="great-filter" name="great-filter" checked> GREAT FC</td>
        </tr><tr>
            <td></td>
			<td><input type="checkbox" id="challenge-filter" name="challenge-filter" checked> CHALLENGE</td>
			<td><input type="checkbox" id="perfect-filter" name="perfect-filter" checked> PERFECT FC</td>
        </tr><tr>
            <td></td>
			<td></td>
			<td><input type="checkbox" id="marvellous-filter" name="marvellous-filter" checked> MARVELLOUS FC</td>
        </tr><tr>
			<td></td>
			<td></td>
			<td><input type="checkbox" id="unplayed-filter" name="unplayed-filter" checked> NOT PLAYED</td>
        </tr>
    </tbody></table>
	</div>
	</div>
	<table id="stats" class="display" style="width:100%">
        <thead>
            <tr>
                <th>Level</th>
                <th>Song Name</th>
                <th>Artist</th>
                <th>Mode</th>
                <th>Difficulty</th>
                <th>Clear Lamp</th>
                <th>Rank</th>
                <th>Score</th>
                <th>Play Count</th>
                <th>Clear Count</th>
                <th>Max Combo</th>
            </tr>
        </thead>
        <tbody>`
	for _, stat := range statsFromServer {
		stats = fmt.Sprintf(`%s
            <tr>
                <td>%d</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
                <td>%d</td>
                <td>%d</td>
                <td>%d</td>
                <td>%d</td>
            </tr>`,
            stats,
            stat.Level,
            stat.Title,
            stat.Artist,
            stat.Mode,
            stat.Difficulty,
            stat.Lamp,
            stat.Rank,
            stat.Score,
            stat.PlayCount,
            stat.ClearCount,
            stat.MaxCombo)
	}
	stats = fmt.Sprintf(`%s
        </tbody>
        <tfoot>
            <tr>
                <th>Level</th>
                <th>Song Name</th>
                <th>Artist</th>
                <th>Mode</th>
                <th>Difficulty</th>
                <th>Clear Lamp</th>
                <th>Rank</th>
                <th>Score</th>
                <th>Play Count</th>
                <th>Clear Count</th>
                <th>Max Combo</th>
            </tr>
        </tfoot>
    </table>`, stats)

	return
}