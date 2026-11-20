package com.fudia.pay.features.sync;

import android.content.Context;
import androidx.work.WorkerParameters;
import dagger.internal.DaggerGenerated;
import dagger.internal.QualifierMetadata;
import dagger.internal.ScopeMetadata;
import javax.annotation.processing.Generated;
import javax.inject.Provider;

@ScopeMetadata
@QualifierMetadata
@DaggerGenerated
@Generated(
    value = "dagger.internal.codegen.ComponentProcessor",
    comments = "https://dagger.dev"
)
@SuppressWarnings({
    "unchecked",
    "rawtypes",
    "KotlinInternal",
    "KotlinInternalInJava",
    "cast",
    "deprecation"
})
public final class SyncWorker_Factory {
  private final Provider<PaymentSyncer> syncerProvider;

  public SyncWorker_Factory(Provider<PaymentSyncer> syncerProvider) {
    this.syncerProvider = syncerProvider;
  }

  public SyncWorker get(Context context, WorkerParameters params) {
    return newInstance(context, params, syncerProvider.get());
  }

  public static SyncWorker_Factory create(Provider<PaymentSyncer> syncerProvider) {
    return new SyncWorker_Factory(syncerProvider);
  }

  public static SyncWorker newInstance(Context context, WorkerParameters params,
      PaymentSyncer syncer) {
    return new SyncWorker(context, params, syncer);
  }
}
