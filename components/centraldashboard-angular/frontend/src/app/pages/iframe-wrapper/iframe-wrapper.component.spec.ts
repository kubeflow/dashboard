import { HttpClientTestingModule } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { SafePipe } from 'src/app/pipes/safe.pipe';
import { MESSAGE } from '../../../../public/library';

import { IframeWrapperComponent } from './iframe-wrapper.component';

describe('IframeWrapperComponent', () => {
  let component: IframeWrapperComponent;
  let fixture: ComponentFixture<IframeWrapperComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RouterTestingModule, HttpClientTestingModule],
      declarations: [IframeWrapperComponent, SafePipe],
    }).compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(IframeWrapperComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should remove the window message listener registered in ngAfterViewInit when destroyed', () => {
    const addSpy = spyOn(window, 'addEventListener').and.callThrough();
    const removeSpy = spyOn(window, 'removeEventListener').and.callThrough();

    const localFixture = TestBed.createComponent(IframeWrapperComponent);
    localFixture.detectChanges();

    const messageCalls = addSpy.calls
      .all()
      .filter(call => call.args[0] === MESSAGE);
    expect(messageCalls.length).toBe(1);
    const handler = messageCalls[0].args[1];

    localFixture.destroy();

    expect(removeSpy).toHaveBeenCalledWith(MESSAGE, handler);
  });
});
