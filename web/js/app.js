// Request wrapper
var req = function(opts) {
	opts.config = function(xhr) {
		xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
	};
	opts.unwrapSuccess = function(response) {
		var response = response || {};
		return response.data;
	};
	opts.unwrapError = function(response) {
		var response = response || {};
		return response.error;
	};

	return m.request(opts);
};

var user = req({
	method: "GET",
	url: "/api/v1/user",
});

var contactLink = req({
	method: "GET",
	url: "/api/v1/contact_link",
});

// Pages

var PageWrapper = function(page) {
	this.controller = function() {};
	this.view = function() {
		return m("div[class='row']",
			m("div[class='twelve columns']", [
				m("div#sidenav[class='three columns']", SideNav),
				m("div[class='nine columns']", page),
			]));
	}
}

var HomePage = {
	view: function() {
		var userInfo = user();
		var contactLinkAddr = contactLink();
		if (!userInfo || !userInfo.name) {
			window.location = '/login';
		}
		return m("div", [
			m("h3", "Profile"),
			m("p", [m("strong", "Main email address:"), m("span", " " + userInfo.mainEmail)]),
			m("p", [
				m("strong", "OneContactLink:"),
				m("span", " "),
				m("a", {href: contactLinkAddr}, contactLinkAddr)
			])
		]);
	}
};

var EmailsPage = {
	view: function() {
		return EmailWidget;
	}
};

// Nav

var TopNav = {
	view: function() {
		return [
			m("li[class='navbar-item']",
				m("a[href='/']", "Home"),
				m("a[href='/']", {config: m.route}, "Manage"),
				m("a[href='/app/logout']", "Logout")
			)
		];
	}
};

var SideNav = {
	view: function() {
		return m("ul", [
			m("li", m("a[href='/']", {config: m.route}, "Home")),
			m("li", m("a[href='/emails']", {config: m.route}, "Emails")),
		]);
	}
}

m.route.mode = "hash";
m.mount(document.querySelector("#nav"), TopNav);
m.route(document.querySelector("#app"), "/", {
	"/": new PageWrapper(HomePage),
	"/emails": new PageWrapper(EmailsPage),
});
