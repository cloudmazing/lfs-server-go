DROP KEYSPACE cqlr;
CREATE KEYSPACE cqlr WITH replication = 
{ 'class' : 'SimpleStrategy', 'replication_factor' : 1 };
