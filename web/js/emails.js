var Email = function(data) {
	data = data || {};
	this.id = m.prop(data.id || 0);
	this.address = m.prop(data.address || "");
	this.user = m.prop(data.user || "");
	this.status = m.prop(data.status || 0);
	this.created = m.prop(data.created || 0);
	this.updated = m.prop(data.updated || 0);
	this.deleted = m.prop(data.deleted || 0);
}

Email.list = function(data) {
	return req({
		method: "GET",
		url: "/api/v1/emails",
		type: Email,
	});
}

Email.activate = function(email) {
	return req({
		method: "POST",
		url: "/api/v1/emails/" + email.address() + "/send_activation",
		data: email,
	})
}

var EmailWidget = {
	controller: function update() {
		this.emails = Email.list();
		this.activate = function(email) {
			Email.activate(email).then(update.bind(this));
		}.bind(this);
	},
	view: function(ctrl) {
		return m("div", [
			m("h3", "Emails"),
			m.component(EmailList, {emails: ctrl.emails, activate: ctrl.activate}),
		]);
	},
}

var EmailListComponent = function(email, activate) {
	this.controller = function() {
		this.email = email;
		this.activate = activate;
	}

	this.view = function(ctrl) {
		var status = ctrl.email.status();
		switch (status) {
		case 2:
			status = "Active";
			break;
		case 1:
			status = "Pending Activation";
			break;
		case 0:
			status = m("button", {onclick: ctrl.activate.bind(this, ctrl.email)}, "Activate");
			break;
		}

		return m("tr", [
			m("td",
				m("span", {style: {fontSize: "1.3em"}}, ctrl.email.address())
			),
			m("td", status),
		])
	}
}

var EmailList = {
	view: function(ctrl, args) {
		return m("table", {style: {width: "100%"}}, [
			m("tr", [
				m("th", "Address"),
				m("th", "Status"),
			]),
			args.emails().map(function(email) {
				return new EmailListComponent(email, args.activate);
			})
		]);
	},
};
