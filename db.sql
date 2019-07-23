CREATE DATABASE bancouati;

/*
Tabela para LOGIN
*/
CREATE TABLE administrators(
  id serial PRIMARY KEY,
  username VARCHAR (50) UNIQUE NOT NULL,
  password VARCHAR (60) NOT NULL,
  created_on TIMESTAMP NOT NULL
);

/*
Tabela dos usuários/funcionários do Banco
*/
CREATE TABLE users(
  id serial PRIMARY KEY,
  name VARCHAR (150),
  email VARCHAR (200) UNIQUE NOT NULL, 
  position VARCHAR (200),
  created_on TIMESTAMP NOT NULL
);  

/*
Tabela de clientes do Banco
*/
CREATE TABLE clients(
  id serial PRIMARY KEY,
  name VARCHAR (150) UNIQUE NOT NULL, 
  salary DECIMAL(10,2), 
  position VARCHAR (150),
  place VARCHAR (200), 
  is_special BOOLEAN NOT NULL DEFAULT false,
  created_on TIMESTAMP NOT NULL
);

/*
Tabela onde fica os funcionários publicos
*/
REATE TABLE public.public_agent (
	"name" varchar(150) NOT NULL,
	"position" varchar(150) NULL,
	place varchar NULL,
	salary numeric NULL,
	CONSTRAINT public_agent_pkey PRIMARY KEY (name)
);


/* 
 Tabela de eventos/notificações
*/
CREATE TABLE public.events (
	id serial NOT NULL,
	qt_leads int4 NULL,
	created_on timestamp NULL,
	CONSTRAINT events_pkey PRIMARY KEY (id)
);

/*
Leads vinculados ao evento
*/
CREATE TABLE public.events_leads (
	id serial NOT NULL,
	"name" varchar(150) NOT NULL,
	event_id int4 NULL,
	created_on timestamp NOT NULL,
	CONSTRAINT events_leads_pkey PRIMARY KEY (id),
	CONSTRAINT events_leads_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id)
);


/*
Usuários que receberam os eventos
*/
CREATE TABLE public.events_to (
	id serial NOT NULL,
	events_id int4 NULL,
	user_id int4 NULL,
	sent_at timestamp NULL,
	CONSTRAINT events_to_pkey PRIMARY KEY (id),
	CONSTRAINT events_to_events_id_fkey FOREIGN KEY (events_id) REFERENCES events(id),
	CONSTRAINT events_to_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id)
);



