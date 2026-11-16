package com.fudia.pay;

import com.fudia.pay.core.network.FudiaPayApi;
import com.fudia.pay.core.security.DeviceCredentials;
import com.fudia.pay.features.sync.SyncScheduler;
import dagger.internal.DaggerGenerated;
import dagger.internal.Factory;
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
public final class SessionViewModel_Factory implements Factory<SessionViewModel> {
  private final Provider<FudiaPayApi> apiProvider;

  private final Provider<DeviceCredentials> credentialsProvider;

  private final Provider<SyncScheduler> schedulerProvider;

  public SessionViewModel_Factory(Provider<FudiaPayApi> apiProvider,
      Provider<DeviceCredentials> credentialsProvider, Provider<SyncScheduler> schedulerProvider) {
    this.apiProvider = apiProvider;
    this.credentialsProvider = credentialsProvider;
    this.schedulerProvider = schedulerProvider;
  }

  @Override
  public SessionViewModel get() {
    return newInstance(apiProvider.get(), credentialsProvider.get(), schedulerProvider.get());
  }

  public static SessionViewModel_Factory create(Provider<FudiaPayApi> apiProvider,
      Provider<DeviceCredentials> credentialsProvider, Provider<SyncScheduler> schedulerProvider) {
    return new SessionViewModel_Factory(apiProvider, credentialsProvider, schedulerProvider);
  }

  public static SessionViewModel newInstance(FudiaPayApi api, DeviceCredentials credentials,
      SyncScheduler scheduler) {
    return new SessionViewModel(api, credentials, scheduler);
  }
}
