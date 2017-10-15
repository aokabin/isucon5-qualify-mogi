package main

import (
	"database/sql"
	"errors"
	"html/template"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
)

var (
	db    *sql.DB
	store *sessions.CookieStore
)

type User struct {
	ID          int
	AccountName string
	NickName    string
	Email       string
}

type Profile struct {
	UserID    int
	FirstName string
	LastName  string
	Sex       string
	Birthday  mysql.NullTime
	Pref      string
	UpdatedAt time.Time
}

type Entry struct {
	ID        int
	UserID    int
	Private   bool
	Title     string
	Content   string
	CreatedAt time.Time
}

type Comment struct {
	ID        int
	EntryID   int
	UserID    int
	Comment   string
	CreatedAt time.Time
}

type Friend struct {
	ID        int
	CreatedAt time.Time
}

type Footprint struct {
	UserID    int
	OwnerID   int
	CreatedAt time.Time
	Updated   time.Time
}

var prefs = []string{"未入力",
	"北海道", "青森県", "岩手県", "宮城県", "秋田県", "山形県", "福島県", "茨城県", "栃木県", "群馬県", "埼玉県", "千葉県", "東京都", "神奈川県", "新潟県", "富山県",
	"石川県", "福井県", "山梨県", "長野県", "岐阜県", "静岡県", "愛知県", "三重県", "滋賀県", "京都府", "大阪府", "兵庫県", "奈良県", "和歌山県", "鳥取県", "島根県",
	"岡山県", "広島県", "山口県", "徳島県", "香川県", "愛媛県", "高知県", "福岡県", "佐賀県", "長崎県", "熊本県", "大分県", "宮崎県", "鹿児島県", "沖縄県"}

var (
	ErrAuthentication   = errors.New("Authentication error.")
	ErrPermissionDenied = errors.New("Permission denied.")
	ErrContentNotFound  = errors.New("Content not found.")
)

var fmap = template.FuncMap{
	"prefectures": func() []string {
		return prefs
	},
	"substring": func(s string, l int) string {
		if len(s) > l {
			return s[:l]
		}
		return s
	},
	"split": strings.Split,
	"getEntry": func(id int) Entry {
		row := db.QueryRow(`SELECT id, user_id, private, body, created_at FROM entries WHERE id=?`, id)
		var entryID, userID, private int
		var body string
		var createdAt time.Time
		checkErr(row.Scan(&entryID, &userID, &private, &body, &createdAt))
		return Entry{id, userID, private == 1, strings.SplitN(body, "\n", 2)[0], strings.SplitN(body, "\n", 2)[1], createdAt}
	},
	"numComments": func(id int) int {
		row := db.QueryRow(`SELECT COUNT(id) AS c FROM comments WHERE entry_id = ?`, id)
		var n int
		checkErr(row.Scan(&n))
		return n
	},
}
