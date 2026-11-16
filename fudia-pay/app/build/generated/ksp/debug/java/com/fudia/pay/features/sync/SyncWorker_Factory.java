package com.fudia.pay.features.sync;

import android.content.Context;
import androidx.work.WorkerParameters;
import com.fudia.pay.core.network.FudiaPayApi;
import com.fudia.pay.features.payments.PendingEventDao;
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
  private final Provider<PendingEventDao> daoProvider;

  private final Provider<FudiaPayApi> apiProvider;

  public SyncWorker_Factory(Provider<PendingEventDao> daoProvider,
      Provider<FudiaPayApi> apiProvider) {
    this.daoProvider = daoProvider;
    this.apiProvider = apiProvider;
  }

  public SyncWorker get(Context context, WorkerParameters params) {
    return newInstance(context, params, daoProvider.get(), apiProvider.get());
  }

  public static SyncWorker_Factory create(Provider<PendingEventDao> daoProvider,
      Provider<FudiaPayApi> apiProvider) {
    return new SyncWorker_Factory(daoProvider, apiProvider);
  }

  public static SyncWorker newInstance(Context context, WorkerParameters params,
      PendingEventDao dao, FudiaPayApi api) {
    return new SyncWorker(context, params, dao, api);
  }
}
