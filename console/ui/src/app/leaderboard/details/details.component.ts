import {AfterViewInit, Component, ElementRef, OnInit, ViewChild} from '@angular/core';
import {Leaderboard} from '../../console.service';
import {ActivatedRoute} from '@angular/router';
import {JSONEditor, Mode} from 'vanilla-jsoneditor';

@Component({
  templateUrl: './details.component.html',
  styleUrls: ['./details.component.scss']
})
export class LeaderboardDetailsComponent implements OnInit, AfterViewInit {
  @ViewChild('editor') private editor: ElementRef<HTMLElement>;

  public orderString = {
    0: 'Ascending',
    1: 'Descending',
  };
  public operatorString = {
    0: 'Best',
    1: 'Set',
    2: 'Increment',
    3: 'Decrement',
  };

  private jsonEditor: JSONEditor;
  public leaderboard: Leaderboard;
  public error = '';

  constructor(private readonly route: ActivatedRoute) {}

  ngOnInit(): void {
    this.route.parent.data.subscribe(
      d => {
        this.leaderboard = d[0];
      },
      err => {
        this.error = err;
      });
  }

  ngAfterViewInit(): void {
    this.jsonEditor = new JSONEditor({
      target: this.editor.nativeElement,
      props: {
        mode: Mode.text,
        readOnly: true,
        content:{text:this.leaderboard.metadata ?? ''},
      },
    });
  }
}
