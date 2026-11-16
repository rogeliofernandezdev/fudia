package com.fudia.pay.features.pairing;

import com.fudia.pay.core.network.FudiaPayApi;
import com.fudia.pay.core.security.DeviceCredentials;
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
public final class PairingViewModel_Factory implements Factory<PairingViewModel> {
  private final Provider<FudiaPayApi> apiProvider;

  private final Provider<DeviceCredentials> credentialsProvider;

  public PairingViewModel_Factory(Provider<FudiaPayApi> apiProvider,
      Provider<DeviceCredentials> credentialsProvider) {
    this.apiProvider = apiProvider;
    this.credentialsProvider = credentialsProvider;
  }

  @Override
  public PairingViewModel get() {
    return newInstance(apiProvider.get(), credentialsProvider.get());
  }

  public static PairingViewModel_Factory create(Provider<FudiaPayApi> apiProvider,
      Provider<DeviceCredentials> credentialsProvider) {
    return new PairingViewModel_Factory(apiProvider, credentialsProvider);
  }

  public static PairingViewModel newInstance(FudiaPayApi api, DeviceCredentials credentials) {
    return new PairingViewModel(api, credentials);
  }
}
