import {Component, OnInit} from '@angular/core';
import {UntypedFormBuilder, UntypedFormGroup} from '@angular/forms';

@Component({
  templateUrl: './notifications-list.component.html',
  styleUrls: ['./notifications-list.component.scss']
})
export class NotificationsListComponent implements OnInit {
  public notificationId: string;
  public searchForm: UntypedFormGroup;

  constructor(
    private readonly formBuilder: UntypedFormBuilder,
  ) {}

  ngOnInit(): void {
    this.searchForm = this.formBuilder.group({
      notification_id: [''],
    });
  }

  search(): void {
    this.notificationId = this.f.notification_id.value;
  }

  get f(): any {
    return this.searchForm.controls;
  }
}
