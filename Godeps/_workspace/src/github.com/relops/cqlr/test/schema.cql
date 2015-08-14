CREATE TABLE tweet
(
    timeline ascii,
    id timeuuid,
    text text,
    PRIMARY KEY (timeline,id)
);

CREATE TABLE tweetcase
(
    "timeLine" ascii,
    timeline ascii,
    id timeuuid,
    text text,
    PRIMARY KEY ("timeLine",id)
);

CREATE TABLE sensors
(    
    id int,
    timestamp timestamp,
    temperature float,
    PRIMARY KEY (id)
);

CREATE TABLE calls
(    
    imsi ascii,
    timestamp timestamp,
    duration int,
    carrier ascii,
    charge decimal,
    PRIMARY KEY (imsi, timestamp)
);

CREATE TABLE queue
(    
    id uuid,
    unix bigint,
    usr ascii,
    msg blob,    
    PRIMARY KEY (id)
);

CREATE TABLE levels
(    
    country text,
    year int,
    height bigint,
    rain int,
    PRIMARY KEY (country, year)
);

CREATE TABLE partial
(    
    id int,
    value ascii,    
    PRIMARY KEY (id)
);

CREATE TABLE key_value
(    
    key uuid,
    value ascii,    
    PRIMARY KEY (key)
);
