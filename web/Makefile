JS_SRC := js/app.js \
          js/emails.js

CSS_SRC := css/style.css

static/js/app.min.js: $(JS_SRC)
	uglifyjs $(JS_SRC) -o $@

static/css/style.min.css: $(CSS_SRC)
	cleancss $(CSS_SRC) -o $@

all: static/js/app.min.js static/css/style.min.css

.PHONY: all
