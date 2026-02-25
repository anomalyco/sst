import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { injectLoad } from '@analogjs/router';
import { toSignal } from '@angular/core/rxjs-interop';

import { load } from './index.server';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [FormsModule],
  template: `
    <form (ngSubmit)="onSubmit($event)">
      <input type="file" name="file">
      <button type="submit">Upload</button>
    </form>
  `,
})
export default class HomeComponent {
  data = toSignal(injectLoad<typeof load>(), { requireSync: true });

  async onSubmit(event: Event): Promise<void> {
    const file = (event.target as HTMLFormElement)['file'].files?.[0]!;

    const image = await fetch(this.data().url, {
      body: file,
      method: 'PUT',
      headers: {
        'Content-Type': file.type,
        'Content-Disposition': `attachment; filename="${file.name}"`,
      },
    });

    window.location.href = image.url.split('?')[0];
  }
}
