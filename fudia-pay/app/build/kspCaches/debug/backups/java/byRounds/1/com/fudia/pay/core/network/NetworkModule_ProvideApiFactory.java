package com.fudia.pay.core.network;

import com.fudia.pay.core.security.DeviceCredentials;
import dagger.internal.DaggerGenerated;
import dagger.internal.Factory;
import dagger.internal.Preconditions;
import dagger.internal.QualifierMetadata;
import dagger.internal.ScopeMetadata;
import javax.annotation.processing.Generated;
import javax.inject.Provider;

@ScopeMetadata("javax.inject.Singleton")
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
public final class NetworkModule_ProvideApiFactory implements Factory<FudiaPayApi> {
  private final Provider<DeviceCredentials> credentialsProvider;

  public NetworkModule_ProvideApiFactory(Provider<DeviceCredentials> credentialsProvider) {
    this.credentialsProvider = credentialsProvider;
  }

  @Override
  public FudiaPayApi get() {
    return provideApi(credentialsProvider.get());
  }

  public static NetworkModule_ProvideApiFactory create(
      Provider<DeviceCredentials> credentialsProvider) {
    return new NetworkModule_ProvideApiFactory(credentialsProvider);
  }

  public static FudiaPayApi provideApi(DeviceCredentials credentials) {
    return Preconditions.checkNotNullFromProvides(NetworkModule.INSTANCE.provideApi(credentials));
  }
}
