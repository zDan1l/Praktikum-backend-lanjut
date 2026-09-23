package model

import "time"

type Comment struct {
	ID        int       `json:"id"`
	ArticleID int       `json:"article_id"`
	AuthorID  int       `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type CommentRequest struct {
	Content string `json:"content"`
}
