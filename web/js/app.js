var user = m.request({
	method: "GET",
	url: "/api/v1/user",
	unwrapSuccess: function(response) {
		return response.data;
	},
	unwrapError: function(response) {
		return response.error;
	},
	config: function(xhr) {
		xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
	}
});

var emails = m.request({
	method: "GET",
	url: "/api/v1/emails",
	unwrapSuccess: function(response) {
		return response.data;
	},
	unwrapError: function(response) {
		return response.error;
	},
	config: function(xhr) {
		xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
	}
});

var contactLink = m.request({
	method: "GET",
	url: "/api/v1/contactLink",
	unwrapSuccess: function(response) {
		return response.data;
	},
	unwrapError: function(response) {
		return response.error;
	},
	config: function(xhr) {
		xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
	}
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
			])
		]);
	}
};

var login = {
	view: function() {
		return m("div", [
			m("form", [
				m("label[for='foo']", "input"),
				m("input[name='foo']")
			])
		]);
	}
};

var nav = {
	view: function() {
		return [
			m("li[class='navbar-item']",
				m("a[href='/']", {config: m.route}, "Home"),
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
