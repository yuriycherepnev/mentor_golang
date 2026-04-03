package main

import (
	"database/sql"
	"fmt"
	"log"
)

func main() {
	dsn := "root:root@tcp(127.0.0.1:3307)/library?charset=utf8mb4&parseTime=true&multiStatements=true"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MySQL")

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS author (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS reader (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS book (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			title VARCHAR(255) DEFAULT NULL,
			id_author INT UNSIGNED NOT NULL,
			id_reader INT UNSIGNED DEFAULT NULL,
			PRIMARY KEY (id),
			KEY idx_id_author (id_author),
			KEY idx_id_reader (id_reader),
			CONSTRAINT fk_id_author_author FOREIGN KEY (id_author) REFERENCES author(id) ON DELETE RESTRICT ON UPDATE CASCADE,
			CONSTRAINT fk_id_reader_reader FOREIGN KEY (id_reader) REFERENCES reader(id) ON DELETE RESTRICT ON UPDATE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}

	for _, migration := range migrations {
		fmt.Println("Migrate up")
		if _, err := db.Exec(migration); err != nil {
			log.Fatalln("Error", err)
		}
	}

	fmt.Println("Insert data")

	data := []string{
		`INSERT INTO author (name) VALUES
			('Лев Толстой'),
			('Фёдор Достоевский'),
			('Джордж Оруэлл')
		ON DUPLICATE KEY UPDATE name = VALUES(name)`,

		`INSERT INTO reader (name) VALUES
			('Иван Иванович'),
			('Петр Петрович')
		ON DUPLICATE KEY UPDATE name = VALUES(name)`,

		`INSERT INTO book (title, id_author, id_reader) VALUES
			('Война и мир', 1, NULL),
			('Преступление и наказание', 2, 1),
			('1984', 2, 1)
		ON DUPLICATE KEY UPDATE title = VALUES(title)`,
	}

	for _, data := range data {
		if _, err := db.Exec(data); err != nil {
			log.Println("Error", err)
		}
	}

	fmt.Println("Migration complete")
}
