import {Component} from 'angular2/core';
import {RouteConfig, ROUTER_DIRECTIVES} from 'angular2/router';

import {HomeComponent} from './HomeComponent'
import {ManageComponent} from './ManageComponent'

@Component({
  // Declare the tag name in index.html to where the component attaches
  selector: 'main-app',
  // Location of the template for this component
  templateUrl: 'app/main_template.html',
  directives: [ROUTER_DIRECTIVES]
})
@RouteConfig([
  {
    path: "/",
    name: "Home",
    component: HomeComponent,
    useAsDefault: true
  },
  {
    path: "/manage",
    name: "Manage",
    component: ManageComponent
  }
])
export class MainApp {}
