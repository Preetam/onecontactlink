package main

const textEmailTemplate = `%s

Cheers!
https://www.onecontact.link/

Click on the following link to unsubscribe from any future messages:
%%unsubscribe_url%%
`

const htmlEmailTemplate = `
<!DOCTYPE HTML>
<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
body {
  background-color: #eee;
  font-family: sans-serif;
  font-size: 14px;
  line-height: 1.5;
  padding: 1em;
}

a {
  color: #1C90F3;
}

table, tr, td {
  border-spacing: 0;
  padding: 0;
  margin: 0;
}

.container {
  background-color: #fff;
  margin: 0 auto;
  max-width: 95%;
  /*width: 30em;*/
}

.header td {
  text-align: center;
  margin: 0;
  background-color: #1C90F3;
  width: 100%;
  color: white;
  font-size: 20px;
}

.header td a {
  color: white;
  text-decoration: none;
}

td {
  padding: 20px;
}

.footer td {
  text-align: center;
  margin: 0;
  background-color: #666;
  width: 100%;
  color: #ddd;
  font-size: 12px;
  padding: 0px 20px;
}

.footer a {
  color: #ddd;
}

.unsubscribe {
  line-height: 3;
  font-size: 12px;
  text-align: center;
  color: #888;
}

.unsubscribe a {
  color: #888;
}
</style>
</head>

<body style="background-color: #eee;font-family: sans-serif;font-size: 14px;line-height: 1.5;padding: 1em;">
<table class="container" style="border-spacing: 0;padding: 0;margin: 0 auto;background-color: #fff;max-width: 95%;">
  <tr class="header" style="border-spacing: 0;padding: 0;margin: 0;"><td style="border-spacing: 0;padding: 20px;margin: 0;text-align: center;background-color: #1C90F3;width: 100%;color: white;font-size: 20px;"><a href="#" style="color: white;text-decoration: none;">OneContact.Link</a></td></tr>
  <tr style="border-spacing: 0;padding: 0;margin: 0;"><td style="border-spacing: 0;padding: 20px;margin: 0;">%s
  <p>Cheers,<br>OneContactLink</p>
  </td></tr>
  <tr class="footer" style="border-spacing: 0;padding: 0;margin: 0;"><td style="border-spacing: 0;padding: 0px 20px;margin: 0;text-align: center;background-color: #666;width: 100%;color: #ddd;font-size: 12px;">
    <p>
      <a href="https://www.onecontact.link" style="color: #ddd;">Home</a>&nbsp;&middot;&nbsp;
      <a href="https://www.onecontact.link/app" style="color: #ddd;">Manage</a>
    </p>
  </td></tr>
</table>

<div class="unsubscribe" style="line-height: 3;font-size: 12px;text-align: center;color: #888;"><a href="%%unsubscribe_url%%" style="color: #888;">Unsubscribe</a> from these emails.</div>
</body>
</html>
`
