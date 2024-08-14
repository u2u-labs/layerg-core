import {Component, OnInit} from '@angular/core';
import {UntypedFormBuilder, UntypedFormGroup} from '@angular/forms';

@Component({
  templateUrl: './purchases-list.component.html',
  styleUrls: ['./purchases-list.component.scss']
})
export class PurchasesListComponent implements OnInit {
  public transactionId: string;
  public searchForm: UntypedFormGroup;

  constructor(
    private readonly formBuilder: UntypedFormBuilder,
  ) {}

  ngOnInit(): void {
    this.searchForm = this.formBuilder.group({
      transaction_id: [''],
    });
  }

  search(): void {
    this.transactionId = this.f.transaction_id.value;
  }

  get f(): any {
    return this.searchForm.controls;
  }
}
