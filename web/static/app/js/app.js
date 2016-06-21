// Main app
var app = {};

app.controller = function() {}
app.view = function() {
  return m("p", "TODO");
}

// Initialize
m.mount(document.body, {controller: app.controller, view: app.view});
