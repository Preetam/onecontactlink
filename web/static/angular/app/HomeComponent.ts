import {Component} from 'angular2/core';

@Component({
  template: `
    <h2>Home</h2>
    Name: {{name}}
    <br>
    <button (click)="setName()">Set name</button>
  `,
})
export class HomeComponent {
	name: '';

	setName() {
		this.name = 'foo';
	}
}
