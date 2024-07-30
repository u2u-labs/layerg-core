// Copyright 2020 The Nakama Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
