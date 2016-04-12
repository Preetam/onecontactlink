import {Component} from 'angular2/core';

@Component({
  template: `
    <h2>Manage</h2>
    <form>
      <div class="row">
        <div class="six columns">
          <label for="nameInput">Your name</label>
          <input class="u-full-width" type="text" placeholder="Richard Hendricks">
        </div>
        <div class="six columns">
          <label for="exampleEmailInput">Your email</label>
          <input class="u-full-width" type="email" placeholder="test@mailbox.com" id="exampleEmailInput">
        </div>
      </div>
      <div class="g-recaptcha" data-sitekey="6Le3rhoTAAAAAK6pdo1YQSXzhnv7TlLjqtcTOlHU"></div>
      <input class="button-primary" type="submit" value="Submit" style="margin-top: 1rem;">
    </form>
  `,
})
export class ManageComponent {
  constructor() {
    var doc = <HTMLDivElement>document.querySelector("body");
    var script = document.createElement('script');
    script.innerHTML = '';
    script.src = 'https://www.google.com/recaptcha/api.js';
    script.async = true;
    script.defer = true;
    doc.appendChild(script);
  }
}
