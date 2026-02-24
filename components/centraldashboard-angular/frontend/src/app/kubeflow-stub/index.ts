/**
 * Kubeflow library stub for local development.
 * This file provides type definitions and minimal implementations
 * for the kubeflow library components used in this project.
 *
 * In production builds, the actual kubeflow library is copied
 * into node_modules/kubeflow from the kubeflow-common-lib build.
 */

import { Injectable, NgModule } from '@angular/core';
import { MatSnackBar, MatSnackBarModule, MatSnackBarConfig } from '@angular/material/snack-bar';

export enum SnackType {
  Success = 'success',
  Error = 'error',
  Warning = 'warning',
  Info = 'info',
}

export interface SnackBarConfig {
  data: {
    msg: string;
    snackType: SnackType;
  };
}

@Injectable({
  providedIn: 'root',
})
export class SnackBarService {
  constructor(private snackBar: MatSnackBar) {}

  open(config: SnackBarConfig): void {
    const snackBarConfig: MatSnackBarConfig = {
      duration: 5000,
      panelClass: `snack-${config.data.snackType}`,
    };
    this.snackBar.open(config.data.msg, 'Close', snackBarConfig);
  }
}

@NgModule({
  imports: [MatSnackBarModule],
  exports: [MatSnackBarModule],
  providers: [SnackBarService],
})
export class SnackBarModule {}
