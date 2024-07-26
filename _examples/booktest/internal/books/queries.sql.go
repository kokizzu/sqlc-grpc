// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: queries.sql

package books

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

const booksByTags = `-- name: BooksByTags :many
SELECT 
  book_id,
  title,
  name,
  isbn,
  tags
FROM books
LEFT JOIN authors ON books.author_id = authors.author_id
WHERE tags && $1::varchar[]
`

type BooksByTagsRow struct {
	BookID int32
	Title  string
	Name   sql.NullString
	Isbn   string
	Tags   []string
}

func (q *Queries) BooksByTags(ctx context.Context, dollar_1 []string) ([]BooksByTagsRow, error) {
	rows, err := q.db.QueryContext(ctx, booksByTags, pq.Array(dollar_1))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BooksByTagsRow
	for rows.Next() {
		var i BooksByTagsRow
		if err := rows.Scan(
			&i.BookID,
			&i.Title,
			&i.Name,
			&i.Isbn,
			pq.Array(&i.Tags),
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const booksByTitleYear = `-- name: BooksByTitleYear :many
SELECT book_id, author_id, isbn, book_type, title, year, available, tags FROM books
WHERE title = $1 AND year = $2
`

type BooksByTitleYearParams struct {
	Title string
	Year  int32
}

// http: GET /books
func (q *Queries) BooksByTitleYear(ctx context.Context, arg BooksByTitleYearParams) ([]Book, error) {
	rows, err := q.db.QueryContext(ctx, booksByTitleYear, arg.Title, arg.Year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Book
	for rows.Next() {
		var i Book
		if err := rows.Scan(
			&i.BookID,
			&i.AuthorID,
			&i.Isbn,
			&i.BookType,
			&i.Title,
			&i.Year,
			&i.Available,
			pq.Array(&i.Tags),
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const createAuthor = `-- name: CreateAuthor :one
INSERT INTO authors (name) VALUES ($1)
RETURNING author_id, name
`

// http: POST /authors
func (q *Queries) CreateAuthor(ctx context.Context, name string) (Author, error) {
	row := q.db.QueryRowContext(ctx, createAuthor, name)
	var i Author
	err := row.Scan(&i.AuthorID, &i.Name)
	return i, err
}

const createBook = `-- name: CreateBook :one
INSERT INTO books (
    author_id,
    isbn,
    book_type,
    title,
    year,
    available,
    tags
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
RETURNING book_id, author_id, isbn, book_type, title, year, available, tags
`

type CreateBookParams struct {
	AuthorID  int32
	Isbn      string
	BookType  BookType
	Title     string
	Year      int32
	Available time.Time
	Tags      []string
}

// http: POST /books
func (q *Queries) CreateBook(ctx context.Context, arg CreateBookParams) (Book, error) {
	row := q.db.QueryRowContext(ctx, createBook,
		arg.AuthorID,
		arg.Isbn,
		arg.BookType,
		arg.Title,
		arg.Year,
		arg.Available,
		pq.Array(arg.Tags),
	)
	var i Book
	err := row.Scan(
		&i.BookID,
		&i.AuthorID,
		&i.Isbn,
		&i.BookType,
		&i.Title,
		&i.Year,
		&i.Available,
		pq.Array(&i.Tags),
	)
	return i, err
}

const deleteBook = `-- name: DeleteBook :exec
DELETE FROM books
WHERE book_id = $1
`

// http: DELETE /books/{book_id}
func (q *Queries) DeleteBook(ctx context.Context, bookID int32) error {
	_, err := q.db.ExecContext(ctx, deleteBook, bookID)
	return err
}

const getAuthor = `-- name: GetAuthor :one
SELECT author_id, name FROM authors
WHERE author_id = $1
`

// http: GET /authors/{author_id}
func (q *Queries) GetAuthor(ctx context.Context, authorID int32) (Author, error) {
	row := q.db.QueryRowContext(ctx, getAuthor, authorID)
	var i Author
	err := row.Scan(&i.AuthorID, &i.Name)
	return i, err
}

const getBook = `-- name: GetBook :one
SELECT book_id, author_id, isbn, book_type, title, year, available, tags FROM books
WHERE book_id = $1
`

// http: GET /books/{book_id}
func (q *Queries) GetBook(ctx context.Context, bookID int32) (Book, error) {
	row := q.db.QueryRowContext(ctx, getBook, bookID)
	var i Book
	err := row.Scan(
		&i.BookID,
		&i.AuthorID,
		&i.Isbn,
		&i.BookType,
		&i.Title,
		&i.Year,
		&i.Available,
		pq.Array(&i.Tags),
	)
	return i, err
}

const updateBook = `-- name: UpdateBook :exec
UPDATE books
SET title = $1, tags = $2, book_type = $3
WHERE book_id = $4
`

type UpdateBookParams struct {
	Title    string
	Tags     []string
	BookType BookType
	BookID   int32
}

// http: PUT /books/{book_id}
func (q *Queries) UpdateBook(ctx context.Context, arg UpdateBookParams) error {
	_, err := q.db.ExecContext(ctx, updateBook,
		arg.Title,
		pq.Array(arg.Tags),
		arg.BookType,
		arg.BookID,
	)
	return err
}

const updateBookISBN = `-- name: UpdateBookISBN :exec
UPDATE books
SET isbn = $1
WHERE book_id = $2
`

type UpdateBookISBNParams struct {
	Isbn   string
	BookID int32
}

// http: PATCH /books/{book_id}/isbn
func (q *Queries) UpdateBookISBN(ctx context.Context, arg UpdateBookISBNParams) error {
	_, err := q.db.ExecContext(ctx, updateBookISBN, arg.Isbn, arg.BookID)
	return err
}
