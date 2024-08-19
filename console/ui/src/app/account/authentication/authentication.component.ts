import {Component, OnInit} from '@angular/core';
import {ApiAccount, ConsoleService, ConsoleUnlinkDeviceBody, ConsoleUpdateAccountBody, UserRole} from '../../console.service';
import {ActivatedRoute, Router} from '@angular/router';
import {AuthenticationService} from '../../authentication.service';
import {UntypedFormBuilder, UntypedFormGroup} from '@angular/forms';

@Component({
  templateUrl: './authentication.component.html',
  styleUrls: ['./authentication.component.scss']
})
export class AuthenticationComponent implements OnInit {
  public error = '';
  public account: ApiAccount;
  public accountForm: UntypedFormGroup;
  public updating = false;
  public updated = false;

  constructor(
    private readonly route: ActivatedRoute,
    private readonly router: Router,
    private readonly consoleService: ConsoleService,
    private readonly authService: AuthenticationService,
    private readonly formBuilder: UntypedFormBuilder,
  ) {}

  ngOnInit(): void {
    this.accountForm = this.formBuilder.group({
      email: [''],
      password: [''],
      selected_device_id_index: [''],
    });

    this.route.parent.data.subscribe(
      d => {
        this.account = d[0].account;
        this.f.email.setValue(this.account.email);
        this.f.password.setValue('');
        this.f.selected_device_id_index.setValue(0);

        if (this.account.devices.length === 0) {
          this.f.selected_device_id_index.disable();
        }

        if (!this.updateAllowed()) {
          this.accountForm.disable();
        }
      },
      err => {
        this.error = err;
      });
  }

  updateAccount(): void {
    this.error = '';
    this.updated = false;
    this.updating = true;

    let body: ConsoleUpdateAccountBody = {email: this.f.email.value};
    if (this.f.password.dirty) {
      body.password = this.f.password.value;
    }

    this.consoleService.updateAccount('', this.account.user.id, body).subscribe(d => {
      this.updated = true;
      this.updating = false;
      this.f.password.reset();
    }, err => {
      this.error = err;
      this.updating = false;
    });
  }

  unlinkDeviceId(event): void {
    event.target.disabled = true;
    this.error = '';

    const body: ConsoleUnlinkDeviceBody = {
      device_id: this.account.devices[this.f.selected_device_id_index.value].id,
    };
    this.consoleService.unlinkDevice('', this.account.user.id, body).subscribe(() => {
      this.error = '';
      this.account.devices.splice(this.f.selected_device_id_index.value, 1);
      this.f.selected_device_id_index.setValue(0);
    }, err => {
      this.error = err;
    });
  }

  unlinkCustomID(event): void {
    event.target.disabled = true;
    this.error = '';

    this.consoleService.unlinkCustom('', this.account.user.id).subscribe(() => {
      this.error = '';
      this.account.custom_id = null;
    }, err => {
      this.error = err;
    });
  }

  unlinkFacebook(event): void {
    event.target.disabled = true;
    this.error = '';

    this.consoleService.unlinkFacebook('', this.account.user.id).subscribe(() => {
      this.error = '';
      this.account.user.facebook_id = null;
    }, err => {
      this.error = err;
    });
  }

  unlinkFacebookInstantGames(event): void {
    event.target.disabled = true;
    this.error = '';

    this.consoleService.unlinkFacebookInstantGame('', this.account.user.id).subscribe(() => {
      this.error = '';
      this.account.user.facebook_instant_game_id = null;
    }, err => {
      this.error = err;
    });
  }

  unlinkApple(event): void {
    event.target.disabled = true;
    this.error = '';

    this.consoleService.unlinkApple('', this.account.user.id).subscribe(() => {
      this.error = '';
      this.account.user.apple_id = null;
    }, err => {
      this.error = err;
    });
  }

  unlinkGameCenter(event): void {
    event.target.disabled = true;
    this.error = '';

    this.consoleService.unlinkGameCenter('', this.account.user.id).subscribe(() => {
      this.error = '';
      this.account.user.gamecenter_id = null;
    }, err => {
      this.error = err;
    });
  }

  unlinkGoogle(event): void {
    event.target.disabled = true;
    this.error = '';

    this.consoleService.unlinkGoogle('', this.account.user.id).subscribe(() => {
      this.error = '';
      this.account.user.google_id = null;
    }, err => {
      this.error = err;
    });
  }

  unlinkSteam(event): void {
    event.target.disabled = true;
    this.error = '';

    this.consoleService.unlinkSteam('', this.account.user.id).subscribe(() => {
      this.error = '';
      this.account.user.steam_id = null;
    }, err => {
      this.error = err;
    });
  }

  updateAllowed(): boolean {
    return this.authService.sessionRole <= UserRole.USER_ROLE_MAINTAINER;
  }

  copyDeviceIdToClipboard(val: string): void {
    const selBox = document.createElement('textarea');
    selBox.style.position = 'fixed';
    selBox.style.left = '0';
    selBox.style.top = '0';
    selBox.style.opacity = '0';
    selBox.value = this.account.devices[val].id;
    document.body.appendChild(selBox);
    selBox.focus();
    selBox.select();
    document.execCommand('copy');
    document.body.removeChild(selBox);
  }

  get f(): any {
    return this.accountForm.controls;
  }
}
