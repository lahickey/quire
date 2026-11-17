// Mixer.mm - an audio mixer in C++, behind an Objective-C class for the app to hold.
#import <AVFoundation/AVFoundation.h>
#import "Mixer.h"

#include <algorithm>
#include <atomic>
#include <memory>
#include <string>
#include <vector>

#define MIXER_MAX_VOICES 32

namespace mixer {

constexpr float kSilence = 1e-4f;

struct Voice {
    std::string name;
    float gain{1.0f};
    std::atomic<bool> playing{false};

    explicit Voice(std::string n) : name(std::move(n)) {}
};

template <typename T>
T clamp01(T v) noexcept {
    return std::min<T>(std::max<T>(v, 0), 1);
}

class Engine final {
public:
    Voice *add(const std::string &name) {
        if (voices_.size() >= MIXER_MAX_VOICES) {
            return nullptr;
        }
        voices_.push_back(std::make_unique<Voice>(name));
        return voices_.back().get();
    }

    // The gain of everything playing, for the meter.
    float level() const {
        float sum = 0;
        for (const auto &v : voices_) {
            if (v->playing.load()) {
                sum += v->gain;
            }
        }
        return sum < kSilence ? 0 : clamp01(sum / static_cast<float>(voices_.size()));
    }

private:
    std::vector<std::unique_ptr<Voice>> voices_;
};

} // namespace mixer

@interface Mixer () <AVAudioPlayerDelegate>
@property (nonatomic, strong) NSMutableDictionary<NSString *, AVAudioPlayer *> *players;
@end

@implementation Mixer {
    std::unique_ptr<mixer::Engine> _engine;
    BOOL _muted;
}

- (instancetype)init {
    if ((self = [super init])) {
        _engine = std::make_unique<mixer::Engine>();
        _players = [NSMutableDictionary dictionary];
    }
    return self;
}

- (BOOL)load:(NSString *)name error:(NSError **)error {
    NSURL *url = [[NSBundle mainBundle] URLForResource:name withExtension:@"caf"];
    if (url == nil) {
        return NO;
    }
    AVAudioPlayer *player = [[AVAudioPlayer alloc] initWithContentsOfURL:url error:error];
    player.delegate = self;
    self.players[name] = player;
    return _engine->add(std::string(name.UTF8String)) != nullptr;
}

/* Muting keeps what is playing, at no volume. */
- (void)setMuted:(BOOL)muted {
    _muted = muted;
    for (AVAudioPlayer *player in self.players.allValues) {
        player.volume = muted ? 0.0f : 1.0f;
    }
}

- (float)level {
    return _muted ? 0 : _engine->level();
}

- (void)play:(NSString *)name after:(NSTimeInterval)delay {
    __weak Mixer *weak = self;
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(delay * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
        Mixer *mixer = weak;
        @synchronized(mixer) {
            [mixer.players[name] play];
        }
    });
}

- (void)audioPlayerDidFinishPlaying:(AVAudioPlayer *)player successfully:(BOOL)flag {
    NSLog(@"finished %@: %@", player.url.lastPathComponent, flag ? @"ok" : @"cut short");
}

@end
