//go:build darwin

package crypto

/*
#cgo LDFLAGS: -framework Security
#include <Security/Security.h>
#include <string.h>

static int kematian_auth_validate_password(const char *user, const char *password) {
	if (user == NULL || password == NULL || user[0] == '\0' || password[0] == '\0') {
		return 0;
	}
	AuthorizationRef authRef = NULL;
	if (AuthorizationCreate(NULL, kAuthorizationEmptyEnvironment, kAuthorizationFlagDefaults, &authRef) != errAuthorizationSuccess) {
		return 0;
	}

	AuthorizationItem right = { kAuthorizationRuleAuthenticateAsSessionUser, 0, NULL, 0 };
	AuthorizationRights rights = { 1, &right };

	AuthorizationItem creds[2];
	creds[0].name = "username";
	creds[0].valueLength = strlen(user);
	creds[0].value = (void *)user;
	creds[0].flags = 0;
	creds[1].name = "password";
	creds[1].valueLength = strlen(password);
	creds[1].value = (void *)password;
	creds[1].flags = 0;

	AuthorizationEnvironment env = { 2, creds };
	OSStatus status = AuthorizationCopyRights(
		authRef,
		&rights,
		&env,
		kAuthorizationFlagDefaults | kAuthorizationFlagExtendRights | kAuthorizationFlagPreAuthorize,
		NULL
	);
	AuthorizationFree(authRef, kAuthorizationFlagDefaults);
	return status == errAuthorizationSuccess;
}
*/
import "C"

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"unsafe"
)

func validateMacLoginPasswordDSCL(password string) bool {
	username := macLoginUsername()
	if username == "" {
		return false
	}
	for _, node := range []string{".", "/Local/Default", "/Search"} {
		cmd := exec.Command("dscl", node, "-authonly", username, password)
		if err := cmd.Run(); err == nil {
			return true
		}
	}
	return false
}

func validateMacLoginPasswordAuthorization(password string) bool {
	username := macLoginUsername()
	if username == "" {
		return false
	}
	cu := C.CString(username)
	cp := C.CString(password)
	defer C.free(unsafe.Pointer(cu))
	defer C.free(unsafe.Pointer(cp))
	return C.kematian_auth_validate_password(cu, cp) == 1
}

// ValidateMacLoginPassword checks the macOS login password without lock/unlock-keychain,
// which would trigger the real Keychain Access system dialog from an untrusted binary.
func ValidateMacLoginPassword(password string) error {
	password = strings.TrimSpace(password)
	if password == "" {
		return fmt.Errorf("empty password")
	}
	if validateMacLoginPasswordAuthorization(password) || validateMacLoginPasswordDSCL(password) {
		return nil
	}
	return fmt.Errorf("invalid macOS login password")
}

func macLoginUsername() string {
	username := strings.TrimSpace(os.Getenv("USER"))
	if username == "" {
		if u, err := user.Current(); err == nil {
			username = strings.TrimSpace(u.Username)
		}
	}
	return username
}