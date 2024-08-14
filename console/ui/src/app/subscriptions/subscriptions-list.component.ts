import {Component, OnInit} from '@angular/core';
import {UntypedFormBuilder, UntypedFormGroup} from '@angular/forms';

@Component({
  templateUrl: './subscriptions-list.component.html',
  styleUrls: ['./subscriptions-list.component.scss']
})
export class SubscriptionsListComponent implements OnInit {
  public originalTransactionId: string;
  public searchForm: UntypedFormGroup;

  constructor(
    private readonly formBuilder: UntypedFormBuilder,
  ) {}

  ngOnInit(): void {
    this.searchForm = this.formBuilder.group({
      original_transaction_id: [''],
    });
  }

  search(): void {
    this.originalTransactionId = this.f.original_transaction_id.value;
  }

  get f(): any {
    return this.searchForm.controls;
  }
}
