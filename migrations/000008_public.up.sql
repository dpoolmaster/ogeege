CREATE TABLE public.articles (
    id           uuid PRIMARY KEY,
    slug         text NOT NULL,
    title        text NOT NULL,
    excerpt      text NOT NULL DEFAULT '',
    content_md   text NOT NULL,
    status       text NOT NULL,
    published_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT articles_status_check CHECK (status IN ('draft', 'published', 'archived'))
);

CREATE UNIQUE INDEX articles_slug_unique ON public.articles (slug);
CREATE INDEX articles_published_idx ON public.articles (status, published_at DESC);

CREATE TABLE public.leads (
    id         uuid PRIMARY KEY,
    name       text NOT NULL,
    contact    text NOT NULL,
    comment    text NOT NULL DEFAULT '',
    source     text NOT NULL DEFAULT '',
    status     text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT leads_status_check CHECK (status IN ('new', 'in_progress', 'processed', 'rejected'))
);

CREATE INDEX leads_created_idx ON public.leads (created_at DESC);

CREATE TABLE public.seo_pages (
    id             uuid PRIMARY KEY,
    path           text NOT NULL,
    title          text NOT NULL,
    description    text NOT NULL DEFAULT '',
    og_title       text NOT NULL DEFAULT '',
    og_description text NOT NULL DEFAULT '',
    og_image       text NOT NULL DEFAULT '',
    canonical_path text NOT NULL DEFAULT '',
    noindex        boolean NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX seo_pages_path_unique ON public.seo_pages (path);
CREATE INDEX seo_pages_path_idx ON public.seo_pages (path);

CREATE TABLE public.redirects (
    id          uuid PRIMARY KEY,
    from_path   text NOT NULL,
    to_path     text NOT NULL,
    status_code integer NOT NULL DEFAULT 301,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT redirects_status_code_check CHECK (status_code IN (301, 302, 307, 308))
);

CREATE UNIQUE INDEX redirects_from_path_unique ON public.redirects (from_path);
CREATE INDEX redirects_from_path_idx ON public.redirects (from_path);
