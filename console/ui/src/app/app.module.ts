import {BrowserModule} from '@angular/platform-browser';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';
import {NgModule} from '@angular/core';

import {AppRoutingModule} from './app-routing.module';
import {AppComponent} from './app.component';
import {HTTP_INTERCEPTORS, HttpClientModule} from '@angular/common/http';
import {WINDOW_PROVIDERS} from './window.provider';
import {environment, environmentLayerg} from '../environments/environment';
import {NgxChartsModule} from '@swimlane/ngx-charts';
import {NgbModule} from '@ng-bootstrap/ng-bootstrap';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {FormsModule, ReactiveFormsModule} from '@angular/forms';
import {NgSelectModule} from '@ng-select/ng-select';
import {Globals} from './globals';
import {SegmentModule} from 'ngx-segment-analytics';
import {SessionInterceptor} from './session.interceptor';
import {AuthenticationErrorInterceptor} from './authentication-error.interceptor';
import {LoginComponent} from './login/login.component';
import {BaseComponent} from './base/base.component';
import {SortNumbersPipe, StatusComponent} from './status/status.component';
import {ConfigComponent} from './config/config.component';
import {ConfigParams} from './console.service';
import {UsersComponent} from './users/users.component';
import {NgxFileDropModule} from 'ngx-file-drop';
import {RuntimeComponent} from './runtime/runtime.component';
import {StorageListComponent} from './storage/storage.component';
import {StorageObjectComponent} from './storage-object/storage-object.component';
import {AccountListComponent} from './accounts/accounts.component';
import {AccountComponent} from './account/account.component';
import {GroupListComponent} from './groups/groups.component';
import {GroupComponent} from './group/group.component';
import {ProfileComponent} from './account/profile/profile.component';
import {GroupDetailsComponent} from './group/details/groupDetailsComponent';
import {AuthenticationComponent} from './account/authentication/authentication.component';
import {FriendsComponent} from './account/friends/friends.component';
import {WalletComponent} from './account/wallet/wallet.component';
import {GroupsComponent} from './account/groups/groups.component';
import {GroupMembersComponent} from './group/members/groupMembers.component';
import {ChatListComponent} from './channels/chat-list.component';
import {PurchasesListComponent} from './purchases/purchases-list.component';
import {MatchesComponent} from './matches/matches.component';
import {LeaderboardsComponent} from './leaderboards/leaderboards.component';
import {LeaderboardComponent} from './leaderboard/leaderboard.component';
import {LeaderboardDetailsComponent} from './leaderboard/details/details.component';
import {LeaderboardRecordsComponent} from './leaderboard/records/records.component';
import {ApiExplorerComponent} from './apiexplorer/apiexplorer.component';
import {PurchasesComponent} from './account/purchases/purchases.component';
import {SubscriptionsComponent} from './account/subscriptions/subscriptions.component';
import {DeleteConfirmDialogComponent} from './shared/delete-confirm-dialog/delete-confirm-dialog.component';
import {SubscriptionsListComponent} from './subscriptions/subscriptions-list.component';
import {CollectionsComponent} from './collections/collections.component';
import {CollectionDetailComponent} from './collections/detail/collectionDetail.component';
import {CreateCollectionComponent} from './collections/create/createCollection.component';
import {CollectionComponent} from './collection/collection.component';
import {CollectionDetail1Component} from './collection/detail/collectionDetail.component';
import {LayergPortalConfig} from './layergPortal.service';
import {ModalCreateNftComponent} from './components/collection/modalCreateNft/modal-create-nft.component';
import {ModalUpdateNftComponent} from "./components/collection/modalUpdateNft/modal-update-nft.component";
import {ModalLinkContractComponent} from "./components/collection/modalLinkContract/modal-link-contract.component";

@NgModule({
  declarations: [
    AppComponent,
    SortNumbersPipe,
    BaseComponent,
    LoginComponent,
    StatusComponent,
    ConfigComponent,
    UsersComponent,
    RuntimeComponent,
    StorageListComponent,
    StorageObjectComponent,
    AccountListComponent,
    AccountComponent,
    ProfileComponent,
    AuthenticationComponent,
    WalletComponent,
    FriendsComponent,
    GroupsComponent,
    GroupComponent,
    GroupDetailsComponent,
    GroupMembersComponent,
    MatchesComponent,
    LeaderboardsComponent,
    LeaderboardComponent,
    LeaderboardDetailsComponent,
    LeaderboardRecordsComponent,
    ApiExplorerComponent,
    PurchasesComponent,
    SubscriptionsComponent,
    GroupListComponent,
    ChatListComponent,
    DeleteConfirmDialogComponent,
    PurchasesListComponent,
    SubscriptionsListComponent,
    CollectionsComponent,
    CreateCollectionComponent,
    CollectionComponent,
    CollectionDetailComponent,
    CollectionDetail1Component,
    ModalCreateNftComponent,
    ModalUpdateNftComponent,
    ModalLinkContractComponent,
  ],
  imports: [
    NgxFileDropModule,
    AppRoutingModule,
    BrowserModule,
    BrowserAnimationsModule,
    HttpClientModule,
    NgbModule,
    NgxChartsModule,
    SegmentModule.forRoot({ apiKey: environment.segment_write_key, debug: !environment.production, loadOnInitialization: !environment.nt }),
    NoopAnimationsModule,
    ReactiveFormsModule,
    FormsModule,
    NgSelectModule,
  ],
  providers: [
    WINDOW_PROVIDERS,
    Globals,
    {provide: ConfigParams, useValue: {host: environment.production ? document.location.origin : environment.apiBaseUrl, timeout: 15000}},
    // tslint:disable-next-line:max-line-length
    {provide: LayergPortalConfig, useValue: {host: environmentLayerg.production ? document.location.origin : environmentLayerg.apiBaseUrl, timeout: 15000}},
    {provide: HTTP_INTERCEPTORS, useClass: SessionInterceptor, multi: true},
    {provide: HTTP_INTERCEPTORS, useClass: AuthenticationErrorInterceptor, multi: true}
  ],
  bootstrap: [AppComponent]
})
export class AppModule {

}
