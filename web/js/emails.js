var Email = function(data) {
	data = data || {};
	this.id = m.prop(data.id || 0);
	this.address = m.prop(data.address || "");
	this.user = m.prop(data.user || 0);
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
	});
}

Email.del = function(email) {
	return req({
		method: "DELETE",
		url: "/api/v1/emails/" + email.address(),
	});
}

Email.create = function(email) {
	return req({
		method: "POST",
		url: "/api/v1/emails",
		data: email,
	});
}

var EmailWidget = {
	controller: function update() {
		this.emails = Email.list();
		this.activate = function(email) {
			Email.activate(email).then(update.bind(this));
		}.bind(this);
		this.del = function(email) {
			Email.del(email).then(update.bind(this));
		}.bind(this);
		this.create = function(email) {
			console.log("Email widget create()");
			Email.create(email).then(update.bind(this));
		}.bind(this);
	},
	view: function(ctrl) {
		return m("div", [
			m.component(EmailList, {emails: ctrl.emails, activate: ctrl.activate, del: ctrl.del}),
			m.component(EmailForm, {create: ctrl.create}),
		]);
	},
}

var EmailListComponent = function(email, activate, del) {
	this.controller = function() {
		this.email = email;
		this.activate = activate;
		this.del = del;
	}

	this.view = function(ctrl) {
		var mainEmail = user().mainEmail === ctrl.email.address();

		var status = ctrl.email.status();
		var buttons = [];
		switch (status) {
		case 2:
			status = "Active";
			break;
		case 1:
			status = "Pending Activation";
			break;
		case 0:
			status = "Requires Activation";
			buttons.push(m("button", {
				class: "btn btn-sm btn-primary",
				onclick: ctrl.activate.bind(this, ctrl.email)
			}, "Activate"));
			break;
		}

		if (mainEmail) {
			status += ", Primary";
		} else {
			buttons.push(
				m("button", {
					class: "btn btn-sm btn-danger",
					onclick: ctrl.del.bind(this, ctrl.email)
				}, "Delete")
			);
		}

		return m("tr", [
			m("td",
				m("span", ctrl.email.address())
			),
			m("td", status),
			m("td", buttons),
		])
	}
}

var EmailList = {
	view: function(ctrl, args) {
		return m("div.table-responsive", m("table.table", [
			m("tr", [
				m("th", "Address"),
				m("th", "Status"),
				m("th", "Manage"),
			]),
			args.emails().map(function(email) {
				return new EmailListComponent(email, args.activate, args.del);
			})
		]));
	},
};

var EmailForm = {
	controller: function() {
		this.email = m.prop(new Email());
	},
	view: function(ctrl, args) {
		var email = ctrl.email();
		return m("div", [
			m("h4", "Add Email"),
			m("form",
			{
				onsubmit: function(e) {
					args.create(email);
					console.log("onsubmit");
					this.reset();
					return false;
				},
			},
			[
				m("div.form-group", [
					m("input.form-control", {type: "email", placeholder: "Email address", oninput: m.withAttr("value", email.address)}, ""),
					m("small", {class: "form-text text-muted"}, "You'll have to activate your email."),
				]),
				m("button", {type: "submit", class: "btn btn-primary"}, "Add"),
			]),
		]);
	},
};
