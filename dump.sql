BEGIN TRANSACTION;
CREATE TABLE users (
		id BIGSERIAL PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL
	);
CREATE TABLE events (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		location TEXT NOT NULL,
		dateTime TIMESTAMP NOT NULL,
		user_id BIGINT,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
CREATE TABLE services (
			id BIGSERIAL PRIMARY KEY,
			name TEXT,
			description TEXT,
			price BIGINT,
			duration BIGINT,
			user_id BIGINT, media_urls TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
CREATE TABLE registrations (
			id BIGSERIAL PRIMARY KEY,
			event_id BIGINT,
			user_id BIGINT,
			FOREIGN KEY (event_id) REFERENCES events(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

INSERT INTO users VALUES(1,'my@login.com','$2a$14$Yy6urJTq1r33UkrpGBjXDO70A61pqu0PA24ZfsttYGzSKSlZXGly.');

INSERT INTO services VALUES(1,'test name ','test description',23,200,1,NULL);
INSERT INTO services VALUES(2,'test images','service with some fancy pics',21,123,1,'null');
INSERT INTO services VALUES(3,'test1','desc1',21,123,1,'null');
INSERT INTO services VALUES(4,'qeqweqw','adsasdasd',2,231,1,'null');
INSERT INTO services VALUES(5,'qeqweqw','adsasdasd',2,231,1,'null');
INSERT INTO services VALUES(6,'eqweqw','dasdasda',21,321,1,'null');
INSERT INTO services VALUES(7,'eqweqwqqqqq','dasdasdawwwwwww',2,3211,1,'null');
INSERT INTO services VALUES(8,'qweqweqweqw','qweqeqweqwe',12,1231,1,'["https://res.cloudinary.com/dojylasam/image/upload/v1753644567/IMG_4881.jpg","https://res.cloudinary.com/dojylasam/image/upload/v1753644568/IMG_7166.jpg"]');
INSERT INTO services VALUES(9,'userid test','description',12,4444,1,'["https://res.cloudinary.com/dojylasam/image/upload/v1753646989/photo_2025-01-02_17-08-14.jpg","https://res.cloudinary.com/dojylasam/image/upload/v1753646990/photo_2025-01-03_11-11-59.jpg"]');
INSERT INTO services VALUES(10,'','',0,0,1,'null');
INSERT INTO services VALUES(11,'Mobile test','desc',15,60,1,'["https://res.cloudinary.com/dojylasam/image/upload/v1754933758/IMG_9343.jpg","https://res.cloudinary.com/dojylasam/image/upload/v1754933759/IMG_9340.jpg","https://res.cloudinary.com/dojylasam/image/upload/v1754933760/IMG_9339.jpg"]');
COMMIT;
