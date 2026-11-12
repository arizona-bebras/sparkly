//! Platform-independent enrollment lifecycle. Storage, cryptography and transport
//! are supplied by the application; this module needs neither std nor allocation.

/// Exclusive credential storage. Hold the platform's lock/transaction guard for
/// the entire load_or_enroll call. Missing values are distinct from corruption:
/// malformed or unreadable credentials must return an error, not None.
pub trait CredentialStore {
    type Error;
    /// May be secret bytes or an opaque reference to a non-exportable key.
    type Key;
    type Certificate;

    fn load_key(&mut self) -> Result<Option<Self::Key>, Self::Error>;
    /// Persist the key before enrollment. Failure must preserve existing keys;
    /// never replace a saved key silently.
    fn save_key(&mut self, key: &Self::Key) -> Result<(), Self::Error>;
    fn load_certificate(&mut self) -> Result<Option<Self::Certificate>, Self::Error>;
    /// Publish the validated certificate atomically, preserving previous state
    /// on failure. Interrupted writes must not look like complete credentials.
    fn save_certificate(&mut self, certificate: &Self::Certificate) -> Result<(), Self::Error>;
}

/// Platform-specific crypto, authenticated enrollment and client construction.
/// The backend owns trust configuration, endpoints, time and HTTP/TLS details.
pub trait IdentityBackend {
    type Error;
    type Key;
    type Certificate;
    type Csr;
    type Client;

    fn generate_key(&self) -> Result<Self::Key, Self::Error>;
    /// Sign a CSR without a subject identity: the server assigns CN/SANs.
    fn create_csr(&self, key: &Self::Key) -> Result<Self::Csr, Self::Error>;
    /// Authenticate the server and send only the token and signed CSR, never the
    /// private key. Enforce transport, redirect, timeout and response-size limits.
    fn enroll(&self, csr: &Self::Csr, token: &str) -> Result<Self::Certificate, Self::Error>;
    /// Validate trust, validity, client-auth usage and key correspondence before
    /// returning a client. Do this for both saved and newly issued credentials.
    fn authenticated_client(
        &self,
        key: &Self::Key,
        certificate: &Self::Certificate,
    ) -> Result<Self::Client, Self::Error>;
}

#[derive(Debug, PartialEq, Eq)]
pub enum IdentityError<Storage, Backend> {
    Storage(Storage),
    Backend(Backend),
    MissingKey,
    MissingEnrollmentToken,
}

impl<S: core::fmt::Display, B: core::fmt::Display> core::fmt::Display for IdentityError<S, B> {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::Storage(e) => write!(f, "credential storage: {e}"),
            Self::Backend(e) => write!(f, "identity backend: {e}"),
            Self::MissingKey => f.write_str("saved certificate has no private key"),
            Self::MissingEnrollmentToken => {
                f.write_str("first enrollment requires an enrollment token")
            }
        }
    }
}

impl<S: core::error::Error + 'static, B: core::error::Error + 'static> core::error::Error
    for IdentityError<S, B>
{
    fn source(&self) -> Option<&(dyn core::error::Error + 'static)> {
        match self {
            Self::Storage(e) => Some(e),
            Self::Backend(e) => Some(e),
            _ => None,
        }
    }
}

#[derive(Debug)]
pub struct Identity<Client> {
    pub client: Client,
    pub newly_enrolled: bool,
}

pub type IdentityResult<Client, StoreError, BackendError> =
    Result<Identity<Client>, IdentityError<StoreError, BackendError>>;

/// Reuse valid credentials or enroll with a persisted key. Failed enrollment or
/// validation keeps the key for retry. Invalid saved credentials never trigger
/// automatic enrollment or key replacement. Renewal is a separate operation.
pub fn load_or_enroll<S, B>(
    store: &mut S,
    backend: &B,
    token: &str,
) -> IdentityResult<B::Client, S::Error, B::Error>
where
    S: CredentialStore,
    B: IdentityBackend<Key = S::Key, Certificate = S::Certificate>,
{
    let certificate = store.load_certificate().map_err(IdentityError::Storage)?;
    if let Some(certificate) = certificate {
        let key = store
            .load_key()
            .map_err(IdentityError::Storage)?
            .ok_or(IdentityError::MissingKey)?;
        let client = backend
            .authenticated_client(&key, &certificate)
            .map_err(IdentityError::Backend)?;
        return Ok(Identity {
            client,
            newly_enrolled: false,
        });
    }
    if token.is_empty() {
        return Err(IdentityError::MissingEnrollmentToken);
    }
    let key = match store.load_key().map_err(IdentityError::Storage)? {
        Some(key) => key,
        None => {
            let key = backend.generate_key().map_err(IdentityError::Backend)?;
            store.save_key(&key).map_err(IdentityError::Storage)?;
            key
        }
    };
    let csr = backend.create_csr(&key).map_err(IdentityError::Backend)?;
    let certificate = backend
        .enroll(&csr, token)
        .map_err(IdentityError::Backend)?;
    let client = backend
        .authenticated_client(&key, &certificate)
        .map_err(IdentityError::Backend)?;
    store
        .save_certificate(&certificate)
        .map_err(IdentityError::Storage)?;
    Ok(Identity {
        client,
        newly_enrolled: true,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use core::cell::Cell;

    #[derive(Debug, PartialEq, Eq)]
    enum Failure {
        CorruptKey,
        PersistKey,
        PersistCertificate,
        Enroll,
        InvalidCertificate,
    }

    #[derive(Default)]
    struct Store {
        key: Option<u8>,
        certificate: Option<u8>,
        corrupt_key: bool,
        fail_key: bool,
        fail_certificate: bool,
    }

    impl CredentialStore for Store {
        type Error = Failure;
        type Key = u8;
        type Certificate = u8;
        fn load_key(&mut self) -> Result<Option<u8>, Failure> {
            if self.corrupt_key {
                Err(Failure::CorruptKey)
            } else {
                Ok(self.key)
            }
        }
        fn save_key(&mut self, key: &u8) -> Result<(), Failure> {
            if self.fail_key {
                return Err(Failure::PersistKey);
            }
            assert!(self.key.is_none(), "never replace an existing key");
            self.key = Some(*key);
            Ok(())
        }
        fn load_certificate(&mut self) -> Result<Option<u8>, Failure> {
            Ok(self.certificate)
        }
        fn save_certificate(&mut self, certificate: &u8) -> Result<(), Failure> {
            if self.fail_certificate {
                return Err(Failure::PersistCertificate);
            }
            self.certificate = Some(*certificate);
            Ok(())
        }
    }

    struct Backend {
        generated: Cell<usize>,
        enrollments: Cell<usize>,
        fail_enrollment: Cell<bool>,
        invalid_certificate: Cell<bool>,
    }
    impl Default for Backend {
        fn default() -> Self {
            Self {
                generated: Cell::new(0),
                enrollments: Cell::new(0),
                fail_enrollment: Cell::new(false),
                invalid_certificate: Cell::new(false),
            }
        }
    }
    impl IdentityBackend for Backend {
        type Error = Failure;
        type Key = u8;
        type Certificate = u8;
        type Csr = u8;
        type Client = u8;
        fn generate_key(&self) -> Result<u8, Failure> {
            self.generated.set(self.generated.get() + 1);
            Ok(7)
        }
        fn create_csr(&self, key: &u8) -> Result<u8, Failure> {
            Ok(*key)
        }
        fn enroll(&self, csr: &u8, token: &str) -> Result<u8, Failure> {
            assert_eq!(token, "token");
            self.enrollments.set(self.enrollments.get() + 1);
            if self.fail_enrollment.get() {
                Err(Failure::Enroll)
            } else {
                Ok(*csr)
            }
        }
        fn authenticated_client(&self, key: &u8, certificate: &u8) -> Result<u8, Failure> {
            if self.invalid_certificate.get() || key != certificate {
                Err(Failure::InvalidCertificate)
            } else {
                Ok(*key)
            }
        }
    }

    #[test]
    fn saved_credentials_skip_enrollment_and_do_not_need_a_token() {
        let mut store = Store {
            key: Some(9),
            certificate: Some(9),
            ..Store::default()
        };
        let backend = Backend::default();
        let identity = load_or_enroll(&mut store, &backend, "").unwrap();
        assert!(!identity.newly_enrolled);
        assert_eq!(identity.client, 9);
        assert_eq!(backend.generated.get(), 0);
        assert_eq!(backend.enrollments.get(), 0);
    }

    #[test]
    fn first_enrollment_requires_token_before_creating_credentials() {
        let mut store = Store::default();
        let backend = Backend::default();
        assert!(matches!(
            load_or_enroll(&mut store, &backend, ""),
            Err(IdentityError::MissingEnrollmentToken)
        ));
        assert_eq!(store.key, None);
        assert_eq!(backend.generated.get(), 0);
        assert_eq!(backend.enrollments.get(), 0);
    }

    #[test]
    fn enrollment_failure_keeps_the_key_for_retry() {
        let mut store = Store::default();
        let backend = Backend::default();
        backend.fail_enrollment.set(true);
        assert!(matches!(
            load_or_enroll(&mut store, &backend, "token"),
            Err(IdentityError::Backend(Failure::Enroll))
        ));
        assert_eq!(store.key, Some(7));
        assert_eq!(store.certificate, None);
        backend.fail_enrollment.set(false);
        assert!(
            load_or_enroll(&mut store, &backend, "token")
                .unwrap()
                .newly_enrolled
        );
        assert_eq!(backend.generated.get(), 1);
        assert_eq!(store.certificate, Some(7));
    }

    #[test]
    fn missing_or_corrupt_saved_keys_never_trigger_reenrollment() {
        let backend = Backend::default();
        let mut store = Store {
            certificate: Some(7),
            ..Store::default()
        };
        assert!(matches!(
            load_or_enroll(&mut store, &backend, "token"),
            Err(IdentityError::MissingKey)
        ));
        store.corrupt_key = true;
        assert!(matches!(
            load_or_enroll(&mut store, &backend, "token"),
            Err(IdentityError::Storage(Failure::CorruptKey))
        ));
        assert_eq!(backend.generated.get(), 0);
        assert_eq!(backend.enrollments.get(), 0);
    }

    #[test]
    fn invalid_certificates_are_not_persisted_or_replaced_silently() {
        let mut store = Store::default();
        let backend = Backend::default();
        backend.invalid_certificate.set(true);
        assert!(matches!(
            load_or_enroll(&mut store, &backend, "token"),
            Err(IdentityError::Backend(Failure::InvalidCertificate))
        ));
        assert_eq!(store.key, Some(7));
        assert_eq!(store.certificate, None);
        store.certificate = Some(7);
        let calls = backend.enrollments.get();
        assert!(load_or_enroll(&mut store, &backend, "token").is_err());
        assert_eq!(backend.enrollments.get(), calls);
        assert_eq!(store.certificate, Some(7));
    }

    #[test]
    fn persistence_failures_do_not_acknowledge_a_complete_identity() {
        let mut store = Store {
            fail_key: true,
            ..Store::default()
        };
        let backend = Backend::default();
        assert!(matches!(
            load_or_enroll(&mut store, &backend, "token"),
            Err(IdentityError::Storage(Failure::PersistKey))
        ));
        assert_eq!(backend.enrollments.get(), 0);
        store.fail_key = false;
        store.fail_certificate = true;
        assert!(matches!(
            load_or_enroll(&mut store, &backend, "token"),
            Err(IdentityError::Storage(Failure::PersistCertificate))
        ));
        assert_eq!(store.key, Some(7));
        assert_eq!(store.certificate, None);
        let generated = backend.generated.get();
        store.fail_certificate = false;
        assert!(
            load_or_enroll(&mut store, &backend, "token")
                .unwrap()
                .newly_enrolled
        );
        assert_eq!(backend.generated.get(), generated);
    }
}
