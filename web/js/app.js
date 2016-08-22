var req = function(opts) {
	opts.config = function(xhr) {
		xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
	};
	opts.unwrapSuccess = function(response) {
		return response.data;
	};
	opts.unwrapError = function(response) {
		return response.error;
	};

	return m.request(opts);
};

var user = req({
	method: "GET",
	url: "/api/v1/user",
});

var emails = req({
	method: "GET",
	url: "/api/v1/emails",
});

var contactLink = req({
	method: "GET",
	url: "/api/v1/contact_link",
});

var home = {
	view: function() {
		var userInfo = user();
		var userEmails = emails();
		var contactLinkAddr = contactLink();
		if (!userInfo || !userInfo.name) {
			window.location = '/login';
		}
		return m("div[class='row']", [
			m("div[class='twelve columns']",
				m("p", "Welcome, " + userInfo.name + ".")
			),
			m("div[class='twelve columns']", [
				m("h5", "Profile"),
				m("p", [m("strong", "Main email address:"), m("span", " " + userInfo.mainEmail)]),
				m("p", [
					m("strong", "OneContactLink:"),
					m("span", " "),
					m("a", {href: contactLinkAddr}, contactLinkAddr)
				])
			]),
			m("ul", [
				emails().map(function(email) {
					return m("li", email.address +
						(userInfo.mainEmail == email.address ? " (main)" : "") +
						(email.status == 0 ? " (pending activation)" : ""));
				})
			]),
			EmailList,
		]);
	}
};

var nav = {
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

m.route.mode = "hash";
m.mount(document.querySelector("#nav"), nav);
m.route(document.querySelector("#app"), "/", {
	"/": home
});
