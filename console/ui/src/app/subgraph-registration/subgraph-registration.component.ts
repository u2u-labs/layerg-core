import {AfterViewInit, Component, Injectable, OnInit} from '@angular/core';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, Router, RouterStateSnapshot} from '@angular/router';
import {ConsoleService} from '../console.service';
import {AuthenticationService} from '../authentication.service';
import {UntypedFormBuilder, UntypedFormGroup, Validators} from '@angular/forms';
import {Observable} from 'rxjs';
import {CHAIN_IDS, parseEventSignaturesOnly} from '../../utils';
import {LayergEventService} from '../layergEvent.service';
import {LayergNodeService} from '../layergNode.service';
import {SubgraphRegistrationData} from '../@types/layergNode';

@Component({
  templateUrl: './subgraph-registration.component.html',
  styleUrls: ['./subgraph-registration.component.scss'],
})
export class SubgraphRegistrationComponent implements OnInit, AfterViewInit {
  constructor(
    private readonly route: ActivatedRoute,
    private readonly router: Router,
    private readonly consoleService: ConsoleService,
    private readonly layergEventService: LayergEventService,
    private readonly layergNodeService: LayergNodeService,
    private readonly authService: AuthenticationService,
    private readonly formBuilder: UntypedFormBuilder,
  ) {}
  public error = '';
  public subgraphForm: UntypedFormGroup;
  public parsedAbiEvent: any;
  public parsedAbi: any;
  public chainIds = CHAIN_IDS;
  public isSuccess = false;

  ngOnInit(): void {
    // Initialization logic here
    const defaultChainId = this.chainIds.length > 0 ? this.chainIds[0].id : null;
    this.subgraphForm = this.formBuilder.group({
      contractAddress: ['', [Validators.required, Validators.minLength(5)]],
      chainId: [defaultChainId, [Validators.required]],
      eventSignature: ['', [Validators.required]],
    });
  }
  get f(): any {
    return this.subgraphForm.controls;
  }
  onFileUpload(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      const file = input.files[0];
      const reader = new FileReader();

      reader.onload = () => {
        try {
           const json = JSON.parse(reader.result as string);
           this.parsedAbiEvent = json.filter((item: any) => item.type === 'event');
           if (!this.parsedAbiEvent || this.parsedAbiEvent.length === 0) {
            console.error('Invalid ABI: No events found');
            return;
           }
           this.parsedAbi = parseEventSignaturesOnly(this.parsedAbiEvent);
        } catch (error) {
          console.error('Invalid JSON file:', error);
        }
      };

      reader.readAsText(file);
    }
  }

  async subgraphRegistration(domainUrl: string, body: SubgraphRegistrationData): Promise<void> {
    try {
      this.layergNodeService.layergNode = {
        host: domainUrl,
        timeoutMs: 5000,
      };
      this.layergNodeService.subgraphRegistration(body).subscribe((d) => {
        console.log(d);
        if (d?.id) {
          this.isSuccess = true;
          this.subgraphForm.reset();
          this.parsedAbiEvent = null;
          this.parsedAbi = null;
        }
      }, err => {
        console.log(err);
        this.error = err;
      });
    } catch (e) {
      this.error = e;
      console.error('Error during subgraph registration:', e);
    }
  }

  async formSubmit(): Promise<void> {
    try {
      this.error = '';
      if (this.subgraphForm.invalid) {
        return;
      }
      if (!this.parsedAbiEvent || this.parsedAbiEvent.length === 0) {
        this.error = 'Invalid ABI: No events found';
        return;
      }
      if (!this.subgraphForm.value.eventSignature) {
        this.error = 'Please upload abi json and select an event signature';
        return;
      }
      const body: SubgraphRegistrationData = {
        contractAddress: this.subgraphForm.value.contractAddress,
        eventSignature: this.subgraphForm.value.eventSignature,
        chainId: Number(this.subgraphForm.value.chainId),
        eventAbi: JSON.stringify(this.parsedAbiEvent),
      };
      const res = await this.layergEventService.getNodeForChainId(this.subgraphForm.value.chainId).toPromise();
      const domainUrl = res?.nodeDomain;
      if (!domainUrl) {
        this.error = 'Node domain not found';
        return;
      }
      console.log('Form submitted', body);
      await this.subgraphRegistration(domainUrl, body);
    } catch (e) {
      this.error = e;
      console.error('Error during form submission:', e);
    }
  }

  ngAfterViewInit(): void {
    // Logic to be executed after the view has been initialized
  }

}

@Injectable({providedIn: 'root'})
export class SubgraphRegistrationResolver implements Resolve<any> {
  constructor(private readonly consoleService: ConsoleService) {}

  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> {
    // const filter = {
    //   limit: route.queryParamMap.get('limit') || 10,
    //   page: route.queryParamMap.get('page') || 1,
    //   search: route.queryParamMap.get('search') || '',
    // };
    // const config = this.consoleService.getConfig('');

    // const collectionList = this.layergPortalService.listCollections(filter);
    return null; // Replace with actual logic to fetch data
  }
}
