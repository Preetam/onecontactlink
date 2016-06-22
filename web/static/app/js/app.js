// Main app
var user = m.request({
    method: "GET",
    url: "/api/v1/user",
    unwrapSuccess: function(response) {
        return response.data;
    },
    unwrapError: function(response) {
        return response.error;
    }
});

var home = {
    view: function() {
        var userInfo = user();
        return m("p", userInfo.name);
    }
}

var login = {
    view: function() {
        return m("div", [
            m("form", [
                m("label[for='foo']", "input"),
                m("input[name='foo']")
            ])
        ]);
    }
}

var nav = {
    view: function() {
        return m("div", [
            m("a[href='/']", {config: m.route}, "home"),
            m("a[href='/login']", {config: m.route}, "login")
        ]);
    }
}

var app = function(page) {
    return {
        view: function() { return [nav, page]; }
    };
}

// Initialize
m.route.mode = "hash";
m.mount(document.querySelector("#nav"), nav);
m.route(document.querySelector("#app"), "/", {
    "/": home,
    "/login": login
});
