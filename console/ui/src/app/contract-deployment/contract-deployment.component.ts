import {Component, Injectable, OnInit, OnDestroy} from '@angular/core';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, Router, RouterStateSnapshot} from '@angular/router';
import {ApiUser, Config, ConsoleService, UserRole} from '../console.service';
import {forkJoin, Observable, Subject} from 'rxjs';
import {UntypedFormBuilder, UntypedFormGroup} from '@angular/forms';
import {AuthenticationService} from '../authentication.service';
import {LayergPortalService} from '../layergPortal.service';
import {NgbModal, NgbModalOptions} from '@ng-bootstrap/ng-bootstrap';
import {ModalLinkContractComponent} from '../components/collection/modalLinkContract/modal-link-contract.component';

@Component({
  templateUrl: './contract-deployment.component.html',
  styleUrls: ['./contract-deployment.component.scss']
})
export class ContractDeploymentComponent implements OnInit, OnDestroy {
  public error = '';
  public contractForm: UntypedFormGroup;

  constructor(
    private readonly route: ActivatedRoute,
    private readonly router: Router,
    private readonly consoleService: ConsoleService,
    private readonly layergPortalService: LayergPortalService,
    private readonly authService: AuthenticationService,
    private readonly formBuilder: UntypedFormBuilder,
    private modalService: NgbModal,
  ) {}

  ngOnInit(): void {
    this.route.data.subscribe(
      d => {
        //
        this.contractForm = this.formBuilder.group({});
      },
      err => {
        this.error = err;
      });
  }

  ngOnDestroy(): void {
    //
  }

  get f(): any {
    return this.contractForm.controls;
  }

  // tslint:disable-next-line:typedef
  // openLinkContractModal(collection: CollectionItem) {
  //   const modalOptions: NgbModalOptions = {
  //     backdrop: true,
  //     centered: true,
  //   };
  //   const modalRef = this.modalService.open(ModalLinkContractComponent, modalOptions);
  //   modalRef.componentInstance.collectionData = collection; // Truyền dữ liệu vào modal
  //   modalRef.componentInstance.linkContractUpdated.subscribe(() => {
  //     this.search();
  //   });
  // }

  handleSubmit(): void {
    // Handle form submission logic here
    console.log(this.contractForm.value);
  }
}

@Injectable({providedIn: 'root'})
export class ContractDeploymentResolver implements Resolve<any> {
  constructor(private readonly consoleService: ConsoleService, private readonly layergPortalService: LayergPortalService) {}

  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> {
    const config = this.consoleService.getConfig('');

    // const collectionList = this.layergPortalService.listCollections(filter);
    return forkJoin([config]);
  }
}
