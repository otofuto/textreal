create table `docs` (
    `id` int auto_increment,
    `title` varchar(255) not null default '',
    `pass` varchar(10) not null default '',
    `text` text not null,
    `token` varchar(255) not null,
    `updated_at` timestamp not null default now(),
    primary key (`id`)
);