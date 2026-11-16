package com.fudia.pay.core.network;

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
public final class AuthInterceptor_Factory implements Factory<AuthInterceptor> {
  private final Provider<DeviceCredentials> credentialsProvider;

  public AuthInterceptor_Factory(Provider<DeviceCredentials> credentialsProvider) {
    this.credentialsProvider = credentialsProvider;
  }

  @Override
  public AuthInterceptor get() {
    return newInstance(credentialsProvider.get());
  }

  public static AuthInterceptor_Factory create(Provider<DeviceCredentials> credentialsProvider) {
    return new AuthInterceptor_Factory(credentialsProvider);
  }

  public static AuthInterceptor newInstance(DeviceCredentials credentials) {
    return new AuthInterceptor(credentials);
  }
}
