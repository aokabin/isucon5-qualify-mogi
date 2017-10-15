package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

func GetLogin(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, "login.html", struct{ Message string }{"高負荷に耐えられるSNSコミュニティサイトへようこそ!"})
}

func PostLogin(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	passwd := r.FormValue("password")
	authenticate(w, r, email, passwd)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func GetLogout(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	delete(session.Values, "user_id")
	session.Options = &sessions.Options{MaxAge: -1}
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func GetIndex(w http.ResponseWriter, r *http.Request) {
	if !authenticated(w, r) {
		return
	}

	user := getCurrentUser(w, r)

	prof := Profile{}
	row := db.QueryRow(`SELECT * FROM profiles WHERE user_id = ?`, user.ID)
	err := row.Scan(&prof.UserID, &prof.FirstName, &prof.LastName, &prof.Sex, &prof.Birthday, &prof.Pref, &prof.UpdatedAt)
	if err != sql.ErrNoRows {
		checkErr(err)
	}

	//rows, err := db.Query(`SELECT id, user_id, private, body, created_at FROM entries WHERE user_id = ? ORDER BY created_at LIMIT 5`, user.ID)
	rows, err := db.Query(`SELECT id, user_id, private, title, created_at FROM entries WHERE user_id = ? ORDER BY created_at LIMIT 5`, user.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	entries := make([]Entry, 0, 5)
	for rows.Next() {
		var id, userID, private int
		var title string
		var createdAt time.Time
		checkErr(rows.Scan(&id, &userID, &private, &title, &createdAt))
		// checkErr(rows.Scan(&id, &userID, &private, &body, &createdAt))
		entries = append(entries, Entry{id, userID, private == 1, title, "", createdAt})
		//entries = append(entries, Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt})
	}
	rows.Close()

	rows, err = db.Query(`SELECT c.id AS id, c.entry_id AS entry_id, c.user_id AS user_id, c.comment AS comment, c.created_at AS created_at
FROM comments c
JOIN entries e ON c.entry_id = e.id
WHERE e.user_id = ?
ORDER BY c.created_at DESC
LIMIT 10`, user.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	commentsForMe := make([]Comment, 0, 10)
	for rows.Next() {
		c := Comment{}
		checkErr(rows.Scan(&c.ID, &c.EntryID, &c.UserID, &c.Comment, &c.CreatedAt))
		commentsForMe = append(commentsForMe, c)
	}
	rows.Close()
	//rows, err = db.Query(`SELECT id, user_id, private, body, created_at FROM entries ORDER BY created_at DESC LIMIT 1000`)
	rows, err = db.Query(`SELECT id, user_id, private, title, created_at FROM entries ORDER BY created_at DESC LIMIT 1000`)
	rows, err = db.Query(`select id, user_id, private, title, created_at from (select id, user_id, private, title, created_at from entries order by created_at desc limit 1000) t1 where user_id in (
		select distinct * from (
		select one as friend from relations where another = ?
		union
		select another as friend from relations where one = ?
		) as b
		) order by created_at desc limit 10;
	`, user.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	entriesOfFriends := make([]Entry, 0, 10)
	for rows.Next() {
		var id, userID, private int
		var title string
		var createdAt time.Time
		checkErr(rows.Scan(&id, &userID, &private, &title, &createdAt))
		/// checkErr(rows.Scan(&id, &userID, &private, &body, &createdAt))
		// if !isFriend(w, r, userID) {
		// 	continue
		// }
		entriesOfFriends = append(entriesOfFriends, Entry{id, userID, private == 1, title, "", createdAt})
		// entriesOfFriends = append(entriesOfFriends, Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt})
		// if len(entriesOfFriends) >= 10 {
		// 	break
		// }
	}
	/*
	   	s := getSession(w, r)
	   	rows, err = db.Query(`SELECT e.id, e.user_id, e.private, e.body, e.created_at FROM entries e
	   WHERE e.user_id IN
	     (select * from (select another from relations where one = ?) as a)
	   ORDER BY e.created_at desc limit 10`, s.Values["user_id"])
	   	if err != sql.ErrNoRows {
	   		checkErr(err)
	   	}
	   	entriesOfFriends := make([]Entry, 0, 10)
	   	for rows.Next() {
	   		var id, userID, private int
	   		var body string
	   		var createdAt time.Time
	   		checkErr(rows.Scan(&id, &userID, &private, &body, &createdAt))
	   		entriesOfFriends = append(entriesOfFriends, Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt})
	   	}
	*/
	rows.Close()

	// rows, err = db.Query(`SELECT id, entry_id, user_id, comment, created_at FROM comments ORDER BY created_at DESC LIMIT 1000`)
	rows, err = db.Query(`SELECT id, entry_id, user_id, comment, created_at FROM comments ORDER BY created_at DESC LIMIT 1000`)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	commentsOfFriends := make([]Comment, 0, 10)
	for rows.Next() {
		c := Comment{}
		checkErr(rows.Scan(&c.ID, &c.EntryID, &c.UserID, &c.Comment, &c.CreatedAt))
		if !isFriend(w, r, c.UserID) {
			continue
		}
		row := db.QueryRow(`SELECT id, user_id, private, title, created_at FROM entries WHERE id = ?`, c.EntryID)
		var id, userID, private int
		var title string
		var createdAt time.Time
		// checkErr(row.Scan(&id, &userID, &private, &body, &createdAt))
		checkErr(row.Scan(&id, &userID, &private, &title, &createdAt))
		entry := Entry{id, userID, private == 1, title, "", createdAt}
		// entry := Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt}
		if entry.Private {
			if !permitted(w, r, entry.UserID) {
				continue
			}
		}
		commentsOfFriends = append(commentsOfFriends, c)
		if len(commentsOfFriends) >= 10 {
			break
		}
	}
	rows.Close()

	rows, err = db.Query(`SELECT * FROM relations WHERE one = ? OR another = ? ORDER BY created_at DESC`, user.ID, user.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	friendsMap := make(map[int]time.Time)
	for rows.Next() {
		var id, one, another int
		var createdAt time.Time
		checkErr(rows.Scan(&id, &one, &another, &createdAt))
		var friendID int
		if one == user.ID {
			friendID = another
		} else {
			friendID = one
		}
		if _, ok := friendsMap[friendID]; !ok {
			friendsMap[friendID] = createdAt
		}
	}
	friends := make([]Friend, 0, len(friendsMap))
	for key, val := range friendsMap {
		friends = append(friends, Friend{key, val})
	}
	rows.Close()

	rows, err = db.Query(`SELECT user_id, owner_id, DATE(created_at) AS date, created_at AS updated
FROM footprints
WHERE user_id = ?
ORDER BY updated DESC
LIMIT 10`, user.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	footprints := make([]Footprint, 0, 10)
	for rows.Next() {
		fp := Footprint{}
		checkErr(rows.Scan(&fp.UserID, &fp.OwnerID, &fp.CreatedAt, &fp.Updated))
		footprints = append(footprints, fp)
	}
	rows.Close()

	render(w, r, http.StatusOK, "index.html", struct {
		User              User
		Profile           Profile
		Entries           []Entry
		CommentsForMe     []Comment
		EntriesOfFriends  []Entry
		CommentsOfFriends []Comment
		Friends           []Friend
		Footprints        []Footprint
	}{
		*user, prof, entries, commentsForMe, entriesOfFriends, commentsOfFriends, friends, footprints,
	})
}

func GetProfile(w http.ResponseWriter, r *http.Request) {
	if !authenticated(w, r) {
		return
	}

	account := mux.Vars(r)["account_name"]
	owner := getUserFromAccount(w, account)
	row := db.QueryRow(`SELECT * FROM profiles WHERE user_id = ?`, owner.ID)
	prof := Profile{}
	err := row.Scan(&prof.UserID, &prof.FirstName, &prof.LastName, &prof.Sex, &prof.Birthday, &prof.Pref, &prof.UpdatedAt)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	var query string
	if permitted(w, r, owner.ID) {
		query = `SELECT id, user_id, private, title, body, created_at FROM entries WHERE user_id = ? ORDER BY created_at LIMIT 5`
	} else {
		query = `SELECT id, user_id, private, title, body, created_at FROM entries WHERE user_id = ? AND private=0 ORDER BY created_at LIMIT 5`
	}
	rows, err := db.Query(query, owner.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	entries := make([]Entry, 0, 5)
	for rows.Next() {
		var id, userID, private int
		var title string
		var body string
		var createdAt time.Time
		checkErr(rows.Scan(&id, &userID, &private, &title, &body, &createdAt))
		// checkErr(rows.Scan(&id, &userID, &private, &body, &createdAt))
		entry := Entry{id, userID, private == 1, title, body, createdAt}
		// entry := Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt}
		entries = append(entries, entry)
	}
	rows.Close()

	markFootprint(w, r, owner.ID)

	render(w, r, http.StatusOK, "profile.html", struct {
		Owner   User
		Profile Profile
		Entries []Entry
		Private bool
	}{
		*owner, prof, entries, permitted(w, r, owner.ID),
	})
}

func PostProfile(w http.ResponseWriter, r *http.Request) {
	if !authenticated(w, r) {
		return
	}
	user := getCurrentUser(w, r)
	account := mux.Vars(r)["account_name"]
	if account != user.AccountName {
		checkErr(ErrPermissionDenied)
	}
	query := `UPDATE profiles
SET first_name=?, last_name=?, sex=?, birthday=?, pref=?, updated_at=CURRENT_TIMESTAMP()
WHERE user_id = ?`
	birth := r.FormValue("birthday")
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	sex := r.FormValue("sex")
	pref := r.FormValue("pref")
	_, err := db.Exec(query, firstName, lastName, sex, birth, pref, user.ID)
	checkErr(err)
	// TODO should escape the account name?
	http.Redirect(w, r, "/profile/"+account, http.StatusSeeOther)
}

func ListEntries(w http.ResponseWriter, r *http.Request) {
	if !authenticated(w, r) {
		return
	}

	account := mux.Vars(r)["account_name"]
	owner := getUserFromAccount(w, account)
	var query string
	if permitted(w, r, owner.ID) {
		query = `SELECT id, user_id, private, title, body, created_at FROM entries WHERE user_id = ? ORDER BY created_at DESC LIMIT 20`
	} else {
		query = `SELECT id, user_id, private, title, body, created_at FROM entries WHERE user_id = ? AND private=0 ORDER BY created_at DESC LIMIT 20`
	}
	rows, err := db.Query(query, owner.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	entries := make([]Entry, 0, 20)
	for rows.Next() {
		var id, userID, private int
		var title string
		var body string
		var createdAt time.Time
		checkErr(rows.Scan(&id, &userID, &private, &title, &body, &createdAt))
		// checkErr(rows.Scan(&id, &userID, &private, &body, &createdAt))
		entry := Entry{id, userID, private == 1, title, body, createdAt}
		// entry := Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt}
		entries = append(entries, entry)
	}
	rows.Close()

	markFootprint(w, r, owner.ID)

	render(w, r, http.StatusOK, "entries.html", struct {
		Owner   *User
		Entries []Entry
		Myself  bool
	}{owner, entries, getCurrentUser(w, r).ID == owner.ID})
}

func GetEntry(w http.ResponseWriter, r *http.Request) {
	if !authenticated(w, r) {
		return
	}
	entryID := mux.Vars(r)["entry_id"]
	row := db.QueryRow(`SELECT id, user_id, private, title, body, created_at FROM entries WHERE id = ?`, entryID)
	var id, userID, private int
	var title string
	var body string
	var createdAt time.Time
	err := row.Scan(&id, &userID, &private, &title, &body, &createdAt)
	// err := row.Scan(&id, &userID, &private, &body, &createdAt)
	if err == sql.ErrNoRows {
		checkErr(ErrContentNotFound)
	}
	checkErr(err)
	entry := Entry{id, userID, private == 1, title, body, createdAt}
	// entry := Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt}
	owner := getUser(w, entry.UserID)
	if entry.Private {
		if !permitted(w, r, owner.ID) {
			checkErr(ErrPermissionDenied)
		}
	}
	rows, err := db.Query(`SELECT * FROM comments WHERE entry_id = ?`, entry.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	comments := make([]Comment, 0, 10)
	for rows.Next() {
		c := Comment{}
		checkErr(rows.Scan(&c.ID, &c.EntryID, &c.UserID, &c.Comment, &c.CreatedAt))
		comments = append(comments, c)
	}
	rows.Close()

	markFootprint(w, r, owner.ID)

	render(w, r, http.StatusOK, "entry.html", struct {
		Owner    *User
		Entry    Entry
		Comments []Comment
	}{owner, entry, comments})
}

func PostEntry(w http.ResponseWriter, r *http.Request) {
	if !authenticated(w, r) {
		return
	}

	user := getCurrentUser(w, r)
	title := r.FormValue("title")
	if title == "" {
		title = "タイトルなし"
	}
	content := r.FormValue("content")
	var private int
	if r.FormValue("private") == "" {
		private = 0
	} else {
		private = 1
	}
	// _, err := db.Exec(`INSERT INTO entries (user_id, private, body) VALUES (?,?,?)`, user.ID, private, title+"\n"+content)
	_, err := db.Exec(`INSERT INTO entries (user_id, private, title, body) VALUES (?,?,?,?)`, user.ID, private, title, content)
	checkErr(err)
	http.Redirect(w, r, "/diary/entries/"+user.AccountName, http.StatusSeeOther)
}

func PostComment(w http.ResponseWriter, r *http.Request) {
	if !authenticated(w, r) {
		return
	}

	entryID := mux.Vars(r)["entry_id"]
	row := db.QueryRow(`SELECT id, user_id, private, title, body, created_at FROM entries WHERE id = ?`, entryID)
	var id, userID, private int
	var title string
	var body string
	var createdAt time.Time
	err := row.Scan(&id, &userID, &private, &title, &body, &createdAt)
	// err := row.Scan(&id, &userID, &private, &body, &createdAt)
	if err == sql.ErrNoRows {
		checkErr(ErrContentNotFound)
	}
	checkErr(err)
	entry := Entry{id, userID, private == 1, title, body, createdAt}
	// entry := Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt}
	owner := getUser(w, entry.UserID)
	if entry.Private {
		if !permitted(w, r, owner.ID) {
			checkErr(ErrPermissionDenied)
		}
	}
	user := getCurrentUser(w, r)
	_, err = db.Exec(`INSERT INTO comments (entry_id, user_id, comment) VALUES (?,?,?)`, entry.ID, user.ID, r.FormValue("comment"))
	checkErr(err)
	http.Redirect(w, r, "/diary/entry/"+strconv.Itoa(entry.ID), http.StatusSeeOther)
}

func GetFootprints(w http.ResponseWriter, r *http.Request) {
	if !authenticated(w, r) {
		return
	}

	user := getCurrentUser(w, r)
	footprints := make([]Footprint, 0, 50)
	rows, err := db.Query(`SELECT user_id, owner_id, DATE(created_at) AS date, created_at as updated
FROM footprints
WHERE user_id = ?
ORDER BY updated DESC
LIMIT 50`, user.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	for rows.Next() {
		fp := Footprint{}
		checkErr(rows.Scan(&fp.UserID, &fp.OwnerID, &fp.CreatedAt, &fp.Updated))
		footprints = append(footprints, fp)
	}
	rows.Close()
	render(w, r, http.StatusOK, "footprints.html", struct{ Footprints []Footprint }{footprints})
}
func GetFriends(w http.ResponseWriter, r *http.Request) {
	if !authenticated(w, r) {
		return
	}

	user := getCurrentUser(w, r)
	rows, err := db.Query(`SELECT * FROM relations WHERE one = ? OR another = ? ORDER BY created_at DESC`, user.ID, user.ID)
	if err != sql.ErrNoRows {
		checkErr(err)
	}
	friendsMap := make(map[int]time.Time)
	for rows.Next() {
		var id, one, another int
		var createdAt time.Time
		checkErr(rows.Scan(&id, &one, &another, &createdAt))
		var friendID int
		if one == user.ID {
			friendID = another
		} else {
			friendID = one
		}
		if _, ok := friendsMap[friendID]; !ok {
			friendsMap[friendID] = createdAt
		}
	}
	rows.Close()
	friends := make([]Friend, 0, len(friendsMap))
	for key, val := range friendsMap {
		friends = append(friends, Friend{key, val})
	}
	render(w, r, http.StatusOK, "friends.html", struct{ Friends []Friend }{friends})
}

func PostFriends(w http.ResponseWriter, r *http.Request) {
	if !authenticated(w, r) {
		return
	}

	user := getCurrentUser(w, r)
	anotherAccount := mux.Vars(r)["account_name"]
	if !isFriendAccount(w, r, anotherAccount) {
		another := getUserFromAccount(w, anotherAccount)
		_, err := db.Exec(`INSERT INTO relations (one, another) VALUES (?,?), (?,?)`, user.ID, another.ID, another.ID, user.ID)
		checkErr(err)
		http.Redirect(w, r, "/friends", http.StatusSeeOther)
	}
}
