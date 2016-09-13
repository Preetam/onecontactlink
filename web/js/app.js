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
		return m("div[class='row']", [
				m("div#sidenav[class='col-sm-3 sidebar-offcanvas']", SideNav),
				"",
				m("div[class='col-sm-9']", page),
			]);
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
			m("li[class='nav-item']", m("a.nav-link[href='/']", "Home")),
			m("li[class='nav-item']", m("a.nav-link[href='/']", {config: m.route}, "Manage")),
			m("li[class='nav-item']", m("a.nav-link[href='/app/logout']", "Logout")),
		];
	}
};

var SideNav = {
	view: function() {
		return m("div", [
			btn("Home", "/"),
			btn("Emails", "/emails")
		]);
		function btn(name, route) {
			var isCurrent = (m.route() === route);
			var click = function(){ m.route(route); };
			return m("a", {
				href: route,
				class: "list-group-item" + (isCurrent ? " active" : ""),
				config: m.route
			}, name);
		}
	}
}

m.route.mode = "hash";
m.mount(document.querySelector("#topnav"), TopNav);
m.route(document.querySelector("#app"), "/", {
	"/": new PageWrapper(HomePage),
	"/emails": new PageWrapper(EmailsPage),
});
