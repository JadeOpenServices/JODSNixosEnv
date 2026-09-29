#include <QCoreApplication>
#include <QTimer>
#include <QUrl>
#include <KIO/FileCopyJob>
#include <KJob>
#include <iostream>
int main(int argc, char **argv) {
    QCoreApplication app(argc, argv);
    if (argc != 3) return 2;
    auto *job = KIO::file_copy(QUrl::fromLocalFile(argv[1]), QUrl::fromLocalFile(argv[2]), -1, KIO::HideProgressInfo);
    job->setUiDelegate(nullptr);
    QObject::connect(job, &KJob::result, &app, [&](KJob *finished) {
        if (finished->error()) std::cerr << finished->errorString().toStdString() << '\n';
        app.exit(finished->error() ? 1 : 0);
    });
    QTimer::singleShot(60000, &app, [&] { job->kill(); app.exit(124); });
    return app.exec();
}
