JS_SRC := js/app.js

CSS_SRC := css/skeleton.css \
           css/style.css

static/js/app.min.js:
	uglifyjs $(JS_SRC) -o $@

static/css/style.min.css:
	cleancss $(CSS_SRC) -o $@

all: static/js/app.min.js static/css/style.min.css

.PHONY: all
