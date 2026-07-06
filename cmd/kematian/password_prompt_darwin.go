//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework AppKit -framework Foundation

#import <AppKit/AppKit.h>
#import <stdlib.h>

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
	@autoreleasepool {
		__block char *result = NULL;
		void (^show)(void) = ^{
			NSString *titleStr = [NSString stringWithUTF8String:(title && title[0]) ? title : "Authentication Required"];
			NSString *msgStr = [NSString stringWithUTF8String:(message && message[0]) ? message : "Enter the password for this Mac to continue."];
			if (show_error) {
				msgStr = [NSString stringWithFormat:@"%@\n\nThe password you entered is incorrect. Please try again.", msgStr];
			}

			NSApplication *app = [NSApplication sharedApplication];
			[app setActivationPolicy:NSApplicationActivationPolicyAccessory];
			[app activateIgnoringOtherApps:YES];

			NSAlert *alert = [[NSAlert alloc] init];
			[alert setMessageText:titleStr];
			[alert setInformativeText:msgStr];
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
			[app setActivationPolicy:NSApplicationActivationPolicyProhibited];
			[app hide:nil];

			if (resp != NSAlertFirstButtonReturn) {
				result = NULL;
				return;
			}

			NSString *pw = [input stringValue];
			if (pw != nil && pw.length > 0) {
				result = strdup([pw UTF8String]);
			}
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

	"recovery/recovery/crypto"
)

var (
	errPasswordPromptCancelled = errors.New("password prompt cancelled")
	errPasswordPromptEmpty     = errors.New("password prompt empty")
)

func showMacPasswordPrompt(title, message string, wrongPassword bool) (string, error) {
	ct := C.CString(title)
	cm := C.CString(message)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cm))

	wrong := C.int(0)
	if wrongPassword {
		wrong = 1
	}

	pw := C.kematian_show_password_dialog(ct, cm, wrong)
	if pw == nil {
		return "", errPasswordPromptCancelled
	}
	defer C.free(unsafe.Pointer(pw))

	s := strings.TrimSpace(C.GoString(pw))
	if s == "" {
		return "", errPasswordPromptEmpty
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

func acquireMacPassword(fromFlag string, noPrompt bool, title, message string, quiet bool) (string, error) {
	_ = quiet
	if p := strings.TrimSpace(fromFlag); p != "" {
		if err := crypto.ValidateMacLoginPassword(p); err != nil {
			return "", err
		}
		return p, nil
	}
	if p := strings.TrimSpace(os.Getenv("KEMATIAN_MAC_PASSWORD")); p != "" {
		if err := crypto.ValidateMacLoginPassword(p); err != nil {
			return "", err
		}
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

	showWrong := false
	for {
		pw, err := showMacPasswordPrompt(title, message, showWrong)
		if err != nil {
			return "", err
		}
		if err := crypto.ValidateMacLoginPassword(pw); err != nil {
			showWrong = true
			continue
		}
		return pw, nil
	}
}